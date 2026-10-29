package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"transcendance/models"
	"transcendance/views"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (pc *Controller) ProfileGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		pc.Renderer.RenderError(c, views.ErrorPage{
			StatusCode: http.StatusUnauthorized,
			Message:    pc.Renderer.T(c, "error-unauthorized"),
		})
		return
	}

	profileUsername := c.Param("username")
	editMode := profileUsername == "" || user.Rank == 2
	if profileUsername == "" {
		profileUsername = user.Username
	}

	var userProfile models.UserProfile
	if err := pc.DB.
		Joins("User", pc.DB.Select("id", "username")).
		Where("User.username = ?", profileUsername).
		First(&userProfile).Error; err != nil {
		pc.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusNotFound,
			Message:    pc.Renderer.T(c, "profile-error-user"),
		})
		return
	}

	builder := views.PageBuilder("base", gin.H{
		"Title": pc.Renderer.T(c, "profile-title"),
		"User":  user,
	})
	formAction := "/profile"
	avatarAction := "/profile/avatar"
	if userProfile.UserID != user.ID {
		formAction = fmt.Sprintf("/profiles/%d", userProfile.UserID)
		avatarAction = formAction + "/avatar"
	}
	builder.Add("profile", "Content", gin.H{
		"User":              user,
		"Profile":           userProfile,
		"Edit":              editMode,
		"CanChangePassword": userProfile.UserID == user.ID,
		"FormAction":        formAction,
		"AvatarAction":      avatarAction,
	})
	pc.Renderer.Render(c, &builder)
}

func (pc *Controller) ProfilePost(c *gin.Context) {
	authUser := GetAuthenticatedUser(c)
	if authUser == nil {
		// Answered in JSON since the profile form submits with fetch
		c.JSON(http.StatusUnauthorized, gin.H{"message": pc.Renderer.T(c, "error-unauthorized")})
		return
	}

	target, status := pc.profileTarget(c, authUser)
	if status != 0 {
		c.JSON(status, gin.H{"message": pc.Renderer.T(c, "error-unauthorized")})
		return
	}
	username, processUsername := c.GetPostForm("username")
	bio, processBio := c.GetPostForm("bio")
	birthDate, processBirthDate := c.GetPostForm("birthdate")
	contactInfo, processContactInfo := c.GetPostForm("contact")
	password, processPassword := c.GetPostForm("password")
	newPassword, processNewPassword := c.GetPostForm("new_password")
	passwordConfirm, processPasswordConfirm := c.GetPostForm("confirm_password")

	if target.ID != authUser.ID && (processPassword || processNewPassword || processPasswordConfirm) {
		c.JSON(http.StatusForbidden, gin.H{"message": pc.Renderer.T(c, "error-unauthorized")})
		return
	}

	updates := map[string]map[string]any{
		"user":    {},
		"profile": {},
	}
	errorMessages := gin.H{
		"username":         "",
		"bio":              "",
		"birthdate":        "",
		"contact":          "",
		"password":         "",
		"new_password":     "",
		"confirm_password": "",
	}
	validRequest := true

	if processUsername && username != target.Username {
		if len(username) < 3 || len(username) > 16 || !is_alphanumeric.MatchString(username) {
			errorMessages["username"] = pc.Renderer.T(c, "profile-error-username")
			validRequest = false
		} else if err := pc.DB.Select("id").Where("username = ? AND id <> ?", username, target.ID).First(&models.User{}).Error; err == nil {
			errorMessages["username"] = pc.Renderer.T(c, "profile-error-username-taken")
			validRequest = false
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": pc.Renderer.T(c, "error-unexpected"),
			})
			return
		}
		updates["user"]["username"] = username
	}

	if processBio {
		if utf8.RuneCountInString(bio) > 244 {
			errorMessages["bio"] = pc.Renderer.T(c, "profile-error-bio")
			validRequest = false
		}
		updates["profile"]["bio"] = bio
	}

	if processBirthDate && birthDate == "" {
		// An empty date clears the birth date
		updates["profile"]["birth_date"] = nil
	} else if processBirthDate {
		d, err := time.Parse("2006-01-02", birthDate)

		if err != nil || d.After(views.LatestToday()) {
			errorMessages["birthdate"] = pc.Renderer.T(c, "profile-error-birthdate")
			validRequest = false
		}
		updates["profile"]["birth_date"] = d
	}

	if processContactInfo {
		if utf8.RuneCountInString(contactInfo) > 256 {
			errorMessages["contact"] = pc.Renderer.T(c, "profile-error-contactinfo")
			validRequest = false
		}
		updates["profile"]["contact_info"] = contactInfo
	}

	if processNewPassword {
		if !processPassword {
			errorMessages["password"] = pc.Renderer.T(c, "profile-error-password-missing")
			validRequest = false
		} else {
			var user models.User
			if err := pc.DB.Select("password_hash").Where("id = ?", target.ID).First(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": pc.Renderer.T(c, "error-unexpected"),
				})
				return
			}
			// Not sanitizing current password for backward-compability
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
				errorMessages["password"] = pc.Renderer.T(c, "profile-error-password")
				validRequest = false
			}
		}

		if len(newPassword) < 8 || len(newPassword) > 72 {
			errorMessages["new_password"] = pc.Renderer.T(c, "profile-error-new_password")
			validRequest = false
		}

		if !processPasswordConfirm {
			errorMessages["confirm_password"] = pc.Renderer.T(c, "profile-error-password-missing")
			validRequest = false
		} else if newPassword != passwordConfirm {
			errorMessages["confirm_password"] = pc.Renderer.T(c, "profile-error-passwords-do-not-match")
			validRequest = false
		}

		if errorMessages["password"] == "" && errorMessages["new_password"] == "" && errorMessages["confirm_password"] == "" {
			newPasswordHash, err := bcrypt.GenerateFromPassword(
				[]byte(newPassword),
				bcrypt.DefaultCost,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": pc.Renderer.T(c, "error-unexpected"),
				})
				return
			}
			updates["user"]["password_hash"] = newPasswordHash
		}
	}

	httpStatusCode := http.StatusOK
	if !validRequest {
		httpStatusCode = http.StatusBadRequest
	} else {
		err := pc.DB.Transaction(func(tx *gorm.DB) error {
			if len(updates["user"]) > 0 {
				if err := tx.Model(&models.User{}).
					Where("id = ?", target.ID).
					Updates(updates["user"]).Error; err != nil {
					return err
				}
			}
			if _, changed := updates["user"]["password_hash"]; changed {
				// Log out everywhere else, the current session stays valid
				token, _ := c.Cookie("session_token")
				if err := tx.Where("user_id = ? AND token <> ?", target.ID, token).
					Delete(&models.UserSession{}).Error; err != nil {
					return err
				}
			}
			if len(updates["profile"]) > 0 {
				if err := tx.Model(&models.UserProfile{}).
					Where("user_id = ?", target.ID).
					Updates(updates["profile"]).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			var mysqlErr *mysql.MySQLError

			// Handling duplicate-username race condition
			if processUsername &&
				username != target.Username &&
				errors.As(err, &mysqlErr) &&
				mysqlErr.Number == 1062 {
				errorMessages["username"] = pc.Renderer.T(c, "profile-error-username-taken")
				c.JSON(http.StatusBadRequest, gin.H{"message": errorMessages})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": pc.Renderer.T(c, "error-unexpected"),
			})
			return
		}
	}
	response := gin.H{"message": errorMessages}
	if validRequest && target.ID != authUser.ID && processUsername && username != target.Username {
		response["profileUrl"] = "/profile/" + username
	}
	c.JSON(httpStatusCode, response)
}

// profileTarget authorizes the target before loading any other account.
func (pc *Controller) profileTarget(c *gin.Context, actor *AuthenticatedUser) (*models.User, int) {
	origin := c.GetHeader("Origin")
	if origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host != c.Request.Host || (u.Scheme != "http" && u.Scheme != "https") {
			return nil, http.StatusForbidden
		}
	}
	if c.GetHeader("Sec-Fetch-Site") == "cross-site" {
		return nil, http.StatusForbidden
	}

	target := &models.User{ID: actor.ID, Username: actor.Username}
	raw := c.Param("id")
	if raw == "" {
		return target, 0
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 || uint64(uint(id)) != id {
		return nil, http.StatusNotFound
	}
	if uint(id) == actor.ID {
		return target, 0
	}
	if actor.Rank != 2 {
		return nil, http.StatusForbidden
	}
	target = &models.User{}
	if err := pc.DB.Select("id", "username").First(target, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound
		}
		return nil, http.StatusInternalServerError
	}
	return target, 0
}
