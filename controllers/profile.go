package controllers

import (
	"errors"
	"net/http"
	"time"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ProfileController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func (pc *ProfileController) ProfileGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		pc.Renderer.RenderError(
			c,
			views.ErrorPage{
				StatusCode: http.StatusUnauthorized,
				Message:    pc.Renderer.T(c, "error-unauthorized"),
			},
		)
		return
	}

	profileUsername := c.Param("username")
	editMode := profileUsername == ""
	if profileUsername == "" {
		profileUsername = user.Username
	}

	var userProfile models.UserProfile
	if err := pc.DB.
		Joins("User").
		Where("User.username = ?", profileUsername).
		First(&userProfile).Error; err != nil {
		pc.Renderer.RenderError(
			c,
			views.ErrorPage{
				User:       user,
				StatusCode: http.StatusNotFound,
				Message:    pc.Renderer.T(c, "profile-error-user"),
			},
		)
		return
	}

	if userProfile.User.Avatar == "" {
		userProfile.User.Avatar = "/static/profile-picture.svg"
	}

	builder := views.PageBuilder("base", gin.H{
		"Title": pc.Renderer.T(c, "profile-title"),
		"User":  user,
	})
	builder.Add("profile", "Content", gin.H{
		"User":    user,
		"Profile": userProfile,
		"Edit":    editMode,
	})
	pc.Renderer.Render(c, &builder)
}

func (pc *ProfileController) ProfilePost(c *gin.Context) {
	authUser := GetAuthenticatedUser(c)
	if authUser == nil {
		pc.Renderer.RenderError(
			c,
			views.ErrorPage{
				User:       nil,
				StatusCode: http.StatusUnauthorized,
				Message:    pc.Renderer.T(c, "error-unauthorized"),
			},
		)
		return
	}

	username, processUsername := c.GetPostForm("username")
	bio, processBio := c.GetPostForm("bio")
	birthDate, processBirthDate := c.GetPostForm("birthdate")
	contactInfo, processContactInfo := c.GetPostForm("contact")
	password, processPassword := c.GetPostForm("password")
	newPassword, processNewPassword := c.GetPostForm("new_password")
	passwordConfirm, processPasswordConfirm := c.GetPostForm("confirm_password")

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

	if processUsername && username != authUser.Username {
		if len(username) < 3 || len(username) > 16 || !is_alphanumeric.MatchString(username) {
			errorMessages["username"] = pc.Renderer.T(c, "profile-error-username")
			validRequest = false
		} else if err := pc.DB.Where("username = ?", username).First(&models.User{}).Error; err == nil {
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
		if len(bio) > 244 {
			errorMessages["bio"] = pc.Renderer.T(c, "profile-error-bio")
			validRequest = false
		}
		updates["profile"]["bio"] = bio
	}

	if processBirthDate {
		d, err := time.Parse("2006-01-02", birthDate)

		if err != nil || d.After(time.Now().UTC()) {
			errorMessages["birthdate"] = pc.Renderer.T(c, "profile-error-birthdate")
			validRequest = false
		}
		updates["profile"]["birth_date"] = d
	}

	if processContactInfo {
		if len(contactInfo) > 256 {
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
			if err := pc.DB.Select("password_hash").Where("id = ?", authUser.ID).First(&user).Error; err != nil {
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
					Where("id = ?", authUser.ID).
					Updates(updates["user"]).Error; err != nil {
					return err
				}
			}
			if len(updates["profile"]) > 0 {
				if err := tx.Model(&models.UserProfile{}).
					Where("user_id = ?", authUser.ID).
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
				username != authUser.Username &&
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
	c.JSON(httpStatusCode, gin.H{"message": errorMessages})
}
