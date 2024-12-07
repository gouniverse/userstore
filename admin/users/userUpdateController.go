package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/gouniverse/form"
	"github.com/gouniverse/hb"
	"github.com/gouniverse/userstore"
	"github.com/gouniverse/userstore/admin/shared"
	"github.com/gouniverse/utils"
	"github.com/samber/lo"
)

// == CONTROLLER ==============================================================

type userUpdateController struct{}

var _ shared.PageInterface = (*userUpdateController)(nil)

// == CONSTRUCTOR =============================================================

func NewUserUpdateController() *userUpdateController {
	return &userUpdateController{}
}

func (c *userUpdateController) ToTag(config shared.Config) hb.TagInterface {
	html, withLayout := c.checkAndProcess(config)

	if !withLayout {
		return hb.Raw(html)
	}

	layout := config.Layout(config.ResponseWriter, config.Request, shared.LayoutOptions{
		Title: `Edit User | User Manager`,
		Body:  html,
		Scripts: []string{
			shared.ScriptHtmx,
			shared.ScriptSwal,
		},
	})

	return hb.Raw(layout)
}

func (controller userUpdateController) checkAndProcess(config shared.Config) (html string, withLayout bool) {
	data, errorMessage := controller.prepareDataAndValidate(config)

	if errorMessage != "" {
		return hb.Div().
			Class("alert alert-danger").
			Text(errorMessage).
			ToHTML(), true
	}

	if config.Request.Method == http.MethodPost {
		return controller.form(data).ToHTML(), false
	}

	return controller.page(data).ToHTML(), true
}

func (controller userUpdateController) page(data userUpdateControllerData) hb.TagInterface {
	breadcrumbs := shared.Breadcrumbs(data.config, []shared.Breadcrumb{
		{
			Name: "Home",
			URL:  shared.Url(data.config.Request, shared.PathHome, nil),
		},
		{
			Name: "User Manager",
			URL:  shared.Url(data.config.Request, shared.PathUsers, nil),
		},
		{
			Name: "Edit User",
			URL: shared.Url(data.config.Request, shared.PathUserUpdate, map[string]string{
				"user_id": data.userID,
			}),
		},
	})

	buttonSave := hb.Button().
		Class("btn btn-primary ms-2 float-end").
		Child(hb.I().Class("bi bi-save").Style("margin-top:-4px;margin-right:8px;font-size:16px;")).
		HTML("Save").
		HxInclude("#FormUserUpdate").
		HxPost(shared.Url(data.config.Request, shared.PathUserUpdate, map[string]string{"user_id": data.userID})).
		HxTarget("#FormUserUpdate")

	buttonCancel := hb.Hyperlink().
		Class("btn btn-secondary ms-2 float-end").
		Child(hb.I().Class("bi bi-chevron-left").Style("margin-top:-4px;margin-right:8px;font-size:16px;")).
		HTML("Back").
		Href(shared.Url(data.config.Request, shared.PathUsers, nil))

	heading := hb.Heading1().
		HTML("Edit User").
		Child(buttonSave).
		Child(buttonCancel)

	card := hb.Div().
		Class("card").
		Child(
			hb.Div().
				Class("card-header").
				Style(`display:flex;justify-content:space-between;align-items:center;`).
				Child(hb.Heading4().
					HTML("User Details").
					Style("margin-bottom:0;display:inline-block;")).
				Child(buttonSave),
		).
		Child(
			hb.Div().
				Class("card-body").
				Child(controller.form(data)))

	userTitle := hb.Heading2().
		Class("mb-3").
		Text("User: ").
		Text(data.user.FirstName()).
		Text(" ").
		Text(data.user.LastName())

	return hb.Div().
		Class("container").
		Child(breadcrumbs).
		Child(hb.HR()).
		Child(heading).
		Child(userTitle).
		Child(card)
}

func (controller userUpdateController) form(data userUpdateControllerData) hb.TagInterface {
	fieldsDetails := []form.FieldInterface{
		form.NewField(form.FieldOptions{
			Label: "Status",
			Name:  "user_status",
			Type:  form.FORM_FIELD_TYPE_SELECT,
			Value: data.formStatus,
			Help:  `The status of the user.`,
			Options: []form.FieldOption{
				{
					Value: "- not selected -",
					Key:   "",
				},
				{
					Value: "Active",
					Key:   userstore.USER_STATUS_ACTIVE,
				},
				{
					Value: "Unverified",
					Key:   userstore.USER_STATUS_UNVERIFIED,
				},
				{
					Value: "Inactive",
					Key:   userstore.USER_STATUS_INACTIVE,
				},
				{
					Value: "In Trash Bin",
					Key:   userstore.USER_STATUS_DELETED,
				},
			},
		}),
		form.NewField(form.FieldOptions{
			Label: "First Name",
			Name:  "user_first_name",
			Type:  form.FORM_FIELD_TYPE_STRING,
			Value: data.formFirstName,
			Help:  `The first name of the user.`,
		}),
		form.NewField(form.FieldOptions{
			Label: "Last Name",
			Name:  "user_last_name",
			Type:  form.FORM_FIELD_TYPE_STRING,
			Value: data.formLastName,
			Help:  `The last name of the user.`,
		}),
		form.NewField(form.FieldOptions{
			Label: "Email",
			Name:  "user_email",
			Type:  form.FORM_FIELD_TYPE_STRING,
			Value: data.formEmail,
			Help:  `The email address of the user.`,
		}),
		form.NewField(form.FieldOptions{
			Label: "Admin Notes",
			Name:  "user_memo",
			Type:  form.FORM_FIELD_TYPE_TEXTAREA,
			Value: data.formMemo,
			Help:  "Admin notes for this bloguser. These notes will not be visible to the public.",
		}),
		form.NewField(form.FieldOptions{
			Label:    "User ID",
			Name:     "user_id",
			Type:     form.FORM_FIELD_TYPE_STRING,
			Value:    data.userID,
			Readonly: true,
			Help:     "The reference number (ID) of the user.",
		}),
		form.NewField(form.FieldOptions{
			Label:    "User ID",
			Name:     "user_id",
			Type:     form.FORM_FIELD_TYPE_HIDDEN,
			Value:    data.userID,
			Readonly: true,
		}),
	}

	formUserUpdate := form.NewForm(form.FormOptions{
		ID: "FormUserUpdate",
	})

	formUserUpdate.SetFields(fieldsDetails)

	if data.formErrorMessage != "" {
		formUserUpdate.AddField(form.NewField(form.FieldOptions{
			Type:  form.FORM_FIELD_TYPE_RAW,
			Value: hb.Swal(hb.SwalOptions{Icon: "error", Text: data.formErrorMessage}).ToHTML(),
		}))
	}

	if data.formSuccessMessage != "" {
		formUserUpdate.AddField(form.NewField(form.FieldOptions{
			Type:  form.FORM_FIELD_TYPE_RAW,
			Value: hb.Swal(hb.SwalOptions{Icon: "success", Text: data.formSuccessMessage}).ToHTML(),
		}))
	}

	return formUserUpdate.Build()
}

func (controller userUpdateController) saveUser(r *http.Request, data userUpdateControllerData) (d userUpdateControllerData, errorMessage string) {
	data.formFirstName = utils.Req(r, "user_first_name", "")
	data.formLastName = utils.Req(r, "user_last_name", "")
	data.formEmail = utils.Req(r, "user_email", "")
	data.formMemo = utils.Req(r, "user_memo", "")
	data.formStatus = utils.Req(r, "user_status", "")

	if data.formStatus == "" {
		data.formErrorMessage = "Status is required"
		return data, ""
	}

	if data.formFirstName == "" {
		data.formErrorMessage = "First name is required"
		return data, ""
	}

	if data.formLastName == "" {
		data.formErrorMessage = "Last name is required"
		return data, ""
	}

	if data.formEmail == "" {
		data.formErrorMessage = "Email is required"
		return data, ""
	}

	if !govalidator.IsEmail(data.formEmail) {
		data.formErrorMessage = "Invalid email address"
		return data, ""
	}

	tokenizedColumns, regularColumns := controller.prepareColumnsForUpdate(data)

	err := controller.saveTokenizedColumns(data, tokenizedColumns)

	if err != nil {
		data.config.Logger.Error("At userUpdateController > prepareDataAndValidate", "error", err.Error())
		data.formErrorMessage = "System error. Saving user failed at tokenized columns"
		return data, ""
	}

	err = controller.saveRegularColumns(data, regularColumns)

	if err != nil {
		data.config.Logger.Error("At userUpdateController > prepareDataAndValidate", "error", err.Error())
		data.formErrorMessage = "System error. Saving user failed at regular columns"
		return data, ""
	}

	// data.user.SetMemo(data.formMemo)
	// data.user.SetStatus(data.formStatus)

	// err := data.config.Store.UserUpdate(context.Background(), data.user)

	// if err != nil {
	// 	data.config.Logger.Error("At userUpdateController > prepareDataAndValidate", "error", err.Error())
	// 	data.formErrorMessage = "System error. Saving user failed"
	// 	return data, ""
	// }

	// err = userTokenize(data.user, data.formFirstName, data.formLastName, data.formEmail)

	if err != nil {
		data.config.Logger.Error("At userUpdateController > prepareDataAndValidate", "error", err.Error())
		data.formErrorMessage = "System error. Saving user failed"
		return data, ""
	}

	data.formSuccessMessage = "User saved successfully"

	return data, ""
}

func (controller userUpdateController) saveTokenizedColumns(data userUpdateControllerData, tokenizedColumns map[string]string) error {
	if tokenizedColumns == nil {
		return errors.New("tokenized columns cannot be nil")
	}

	if len(tokenizedColumns) < 1 {
		return nil // nothing to do
	}

	tokensToCreate := map[string]string{}
	tokensToUpdate := map[string]string{}
	tokensToDelete := []string{}

	currentColumnValues := map[string]string{}

	for _, columnName := range data.config.TokenizedColumns {
		currentColumnValues[columnName] = data.user.Get(columnName)
	}

	for columnName, currentValue := range currentColumnValues {
		newValue := lo.ValueOr(tokenizedColumns, columnName, "")

		isToken := strings.HasPrefix(currentValue, "tk_")

		if isToken {
			token := currentValue
			tokensToUpdate[token] = newValue
		} else {
			tokensToCreate[columnName] = newValue
		}
	}

	allCurrentData := data.user.Data()

	for columnName := range allCurrentData {
		isToken := strings.HasPrefix(columnName, "tk_")

		if !isToken {
			continue // not a token
		}

		if lo.HasKey(tokenizedColumns, columnName) {
			continue // already set to be updated or created
		}

		tokensToDelete = append(tokensToDelete, columnName)
	}

	createdTokens, err := data.config.TokensBulk(tokensToCreate, tokensToUpdate, tokensToDelete)

	if err != nil {
		data.config.Logger.Error("At userUpdateController > saveTokenizedColumns", "error", err.Error())
		return err
	}

	for columnName, createdToken := range createdTokens {
		data.user.Set(columnName, createdToken)
	}

	err = data.config.Store.UserUpdate(context.Background(), data.user)

	if err != nil {
		data.config.Logger.Error("At userUpdateController > saveTokenizedColumns", "error", err.Error())
		return err
	}

	return nil
}

func (controller userUpdateController) saveRegularColumns(data userUpdateControllerData, regularColumns map[string]string) error {
	for key, value := range regularColumns {
		switch key {
		case userstore.COLUMN_FIRST_NAME:
			data.user.SetFirstName(value)
		case userstore.COLUMN_LAST_NAME:
			data.user.SetLastName(value)
		case userstore.COLUMN_EMAIL:
			data.user.SetEmail(value)
		case userstore.COLUMN_STATUS:
			data.user.SetStatus(value)
		case userstore.COLUMN_MEMO:
			data.user.SetMemo(value)
		default:
			data.config.Logger.Error("At userUpdateController > saveRegularColumns", "unknown column", key)
		}
	}

	err := data.config.Store.UserUpdate(context.Background(), data.user)

	if err != nil {
		data.config.Logger.Error("At userUpdateController > saveRegularColumns", "error", err.Error())
		return err
	}

	return nil
}

func (controller userUpdateController) prepareColumnsForUpdate(data userUpdateControllerData) (tokenizedColumns map[string]string, regularColumns map[string]string) {
	allColumnUpdates := map[string]string{}
	allColumnUpdates[userstore.COLUMN_FIRST_NAME] = data.formFirstName
	allColumnUpdates[userstore.COLUMN_LAST_NAME] = data.formLastName
	allColumnUpdates[userstore.COLUMN_EMAIL] = data.formEmail
	allColumnUpdates[userstore.COLUMN_STATUS] = data.formStatus
	allColumnUpdates[userstore.COLUMN_MEMO] = data.formMemo

	keys := lo.Keys(allColumnUpdates)

	tokenizedColumns = map[string]string{}
	regularColumns = map[string]string{}

	for _, key := range keys {
		if slices.Contains(data.config.TokenizedColumns, key) {
			tokenizedColumns[key] = allColumnUpdates[key]
		} else {
			regularColumns[key] = allColumnUpdates[key]
		}
	}

	return tokenizedColumns, regularColumns
}

func (controller userUpdateController) prepareDataAndValidate(config shared.Config) (data userUpdateControllerData, errorMessage string) {
	data.config = config
	data.action = utils.Req(config.Request, "action", "")
	data.userID = utils.Req(config.Request, "user_id", "")

	if data.userID == "" {
		return data, "User ID is required"
	}

	user, err := config.Store.UserFindByID(context.Background(), data.userID)

	if err != nil {
		config.Logger.Error("At userUpdateController > prepareDataAndValidate", "error", err.Error())
		return data, "User not found"
	}

	if user == nil {
		return data, "User not found"
	}

	data.user = user

	// firstName, lastName, email, err := helpers.UserUntokenized(data.user)

	// if err != nil {
	// 	config.Logger.Error("At userManagerController > tableUsers", "error", err.Error())
	// 	return data, "Tokens failed to be read"
	// }

	firstName := data.user.FirstName()
	lastName := data.user.LastName()
	email := data.user.Email()

	data.formFirstName = firstName
	data.formLastName = lastName
	data.formEmail = email
	data.formMemo = data.user.Memo()
	data.formStatus = data.user.Status()

	if config.Request.Method != http.MethodPost {
		return data, ""
	}

	return controller.saveUser(config.Request, data)
}

type userUpdateControllerData struct {
	config shared.Config
	action string
	userID string
	user   userstore.UserInterface

	formErrorMessage   string
	formSuccessMessage string
	formEmail          string
	formFirstName      string
	formLastName       string
	formMemo           string
	formStatus         string
}
