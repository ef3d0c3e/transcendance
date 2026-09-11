# Organization

**General organization**

Github is the main platform to organize the project and keep track of advancement.
Everything that *needs* to be done is a **ticket** (Github issues).
Everything that changes the project should be reviewed through a **pull request**.

The project is organized into **milestones**, milestones are a group of capabilities that the application must fulfill: *authentication, api, ...* \
Each milestone is split into **tickets** (github issues), tickets represent an individual unit of work: *user registration, comments on posts, ...* \
**Tickets** are completed in **pull requests**, then they are merged into the main branch.

**You will do your best job to keep the traceability chain accurate:**
```
Requirement → Issue → Branch → Commits → PR → Review → Tests → Merge
```


## Tickets (issues)

Tickets are Github issues corresponding to features, fixes, and general tasks that have to be done.

When creating a ticket, you must specify which part of the application it applies to in its title.
For instance `auth: registration for users`.
Then you must set an appropriate set of tags for the ticket.

Here's the list of ticket tags:
 * `feature` A new user-facing capability
 * `bug` Something is incorrect/broken
 * `enhancement` Improvement to an existing capability
 * `documentation` Docs/specs/examples
 * `testing` Test/QA work that deserves its own issue
 * `refactor` Internal restructuring with no intended behavior change
 * `chore` Maintenance/tooling/config/CI/deployment work
 * `security` Security-sensitive work
 * `blocked` Cannot progress due to an external/internal dependency
 * `question` Question/Unsure about a feature

As well as the priority labels:
 * `priority:low`
 * `priority:medium`
 * `priority:high`

Feature tickets represent represent the *contract* that has to be implemented.
They should be structured as a list of items needed for completion.

Here's an example ticket:
```
Title
=====
auth: registration for users

Tags
====
feature, priority:medium

Description
===========

Implement the registration service for regular users.

Register form:
 * Username: alphanumerical ascii, case-insensitively unique
 * Email: optional
 * Password: ensure it's at least 12 characters
 * Confirmation password
 * Bot verification
 * TOS checkbox

After successful registration, users are automatically logged-in.
An invalid registration attempt shouldn't reload the entire form, but notify the user of which fields are erreonous.

Other data that must be tracked at registration:
 * Account creation date
 * Account creation IP address

Newly registered users must get a default profile picture
```

And here's what a completion PR would look like for this ticket
```
Title
=====
auth: #71 Implement registration for users

Tags
====
feature

Description
===========

Refs #71

## Implementation

 * [x] Registration form
 * [x] Username uniqueness enforced
 * [x] Email stored when provided
 * [x] Password requirements enforced
 * [x] Password confirmation validated
 * [x] Bot verification
 * [x] TOS acceptance
 * [x] Account creation date recorded
 * [x] Account creation IP recorded
 * [x] Default profile picture assigned
 * [x] User logged in after registration
 * [x] Update database for fields tracked at registration

## Tests

### Automated
 * Valid registration attempt
 * Invalid usernames properly refused
 * A missing field fails registration
 * Mismatching passwords are properly reported

### Manual
 * Created an account manually
 * Errors are displayed without reloading the page

```

## Pull requests

Pull requests should specify which part of the application they apply to, and which features they refer to.

**Examples:**
```
auth: #12 Implement user registration
auth: #89, #91 Implement profiles and friends
```

Issues should follow the same principle:
```
auth: account deletion
api: minimal required methods
```

While working on a feature, it is good practice to open a PR, even when the feature is not finished. You should keep track of advancement using a checklist like so:
```
Title
=====
auth: #71 Implement registration

Description
===========

Refs #71

 * [x] User account registration
 * [x] User login
 * [ ] Admin can register accounts for other users

## Tests

### Automated
 * [x] Valid registration attempt
 * [x] Invalid usernames properly refused
 * [x] A missing field fails registration
 * [ ] Mismatching passwords are properly reported

### Manual
TODO

```

The main purpose of this, is that people may use your branch while still not fully complete, if you have implemented what is required for theirs.

### Definition of Done

A ticket is considered complete when:

 * The requirements in the ticket are implemented.
 * Appropriate automated tests have been added or updated.
 * Existing tests pass.
 * The feature has been manually verified where applicable.
 * The pull request has been reviewed.
 * The pull request is merged into `main`.

## Commits and branches

All commits must be made in their own branch.
Ideally the branch should reflect the ticket they refer to.

For instance suppose these tickets:
```
#71 auth: Implement user registration
#72 auth: Implement user login
```

If you implement ticket `#71`, then your branch name should be `feat/71-user-registration`.
If you implement both, ticket `#71` and ticket `#72`, then the branch name should be `feat/auth-user-account`.

The goal of this naming scheme is to make it easier to see which ticked is closed by which branch.
Although that should be mentionned in commit messages and the pull request.

Follow guidelines from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0).

**Examples:**
```
# Features
# features must be testable, meaning they have to actually implement something.

feat: file upload
feat: delete user account

# Bugfixes
# fixes should be supported by regression tests to make sure the fix is still working

fix: crash when uploading video
# This one fixes an open issue:
fix: #418 prevent regular users from accessing admin panel

# Chores
chore: update postgres version
chore: linter passing
```

Here's the list of commit *types* you should be using:
 * `feat` when implementing something
 * `fix` when fixing a bug
 * `doc` when documenting something
 * `chore` when doing chores
 * `test` when adding tests
 * `refactoring` when refactoring code
 * `style` when adjusting for style guidelines
