package auth

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"

	"adrift-backend/configuration"
	DB "adrift-backend/database"
	Schemas "adrift-backend/database/schemas"
	"adrift-backend/utilities"
)

func signUpController(ctx *fiber.Ctx) error {
	var body SignUpBodyStruct
	bodyParsingError := ctx.BodyParser(&body)
	if bodyParsingError != nil {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.InternalServerError,
			Status: fiber.StatusInternalServerError,
		})
	}

	client := body.Client
	email := body.Email
	firstName := body.FirstName
	lastName := body.LastName
	password := body.Password
	signedAgreement := body.SignedAgreement

	if client == "" || email == "" || firstName == "" ||
		lastName == "" || password == "" || !signedAgreement {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.MissingData,
			Status: fiber.StatusBadRequest,
		})
	}

	trimmedClient := strings.TrimSpace(client)
	trimmedEmail := strings.TrimSpace(email)
	trimmedFirstName := strings.TrimSpace(firstName)
	trimmedLastName := strings.TrimSpace(lastName)
	trimmedPassword := strings.TrimSpace(password)

	if trimmedClient == "" || trimmedEmail == "" || trimmedFirstName == "" ||
		trimmedLastName == "" || trimmedPassword == "" {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.MissingData,
			Status: fiber.StatusBadRequest,
		})
	}

	emailIsValid := utilities.ValidateEmail(trimmedEmail)
	if !emailIsValid {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.InvalidEmail,
			Status: fiber.StatusBadRequest,
		})
	}

	clients := utilities.Values(configuration.Clients)
	if !utilities.IncludesString(clients, trimmedClient) {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.InvalidData,
			Status: fiber.StatusBadRequest,
		})
	}

	UserCollection := DB.Instance.Database.Collection(DB.Collections.User)

	existingRecord := UserCollection.FindOne(
		ctx.Context(),
		bson.D{{Key: "email", Value: trimmedEmail}},
	)
	existingUser := &Schemas.User{}
	existingRecord.Decode(existingUser)
	if existingUser.ID != "" {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.EmailAlreadyInUse,
			Status: fiber.StatusBadRequest,
		})
	}

	now := utilities.MakeTimestamp()
	NewUser := new(Schemas.User)
	NewUser.ID = ""
	NewUser.Email = trimmedEmail
	NewUser.FirstName = trimmedFirstName
	NewUser.LastName = trimmedLastName
	NewUser.Role = configuration.Roles.User
	NewUser.SignedAgreement = true
	NewUser.Created = now
	NewUser.Updated = now
	insertionResult, insertionError := UserCollection.InsertOne(ctx.Context(), NewUser)
	if insertionError != nil {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.InternalServerError,
			Status: fiber.StatusInternalServerError,
		})
	}
	createdRecord := UserCollection.FindOne(
		ctx.Context(),
		bson.D{{Key: "_id", Value: insertionResult.InsertedID}},
	)
	createdUser := &Schemas.User{}
	createdRecord.Decode(createdUser)

	UserSecretCollection := DB.Instance.Database.Collection(DB.Collections.UserSecret)

	secret, secretError := utilities.MakeHash(
		createdUser.ID + fmt.Sprintf("%v", utilities.MakeTimestamp()),
	)
	if secretError != nil {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.InternalServerError,
			Status: fiber.StatusInternalServerError,
		})
	}

	// create a new Image record and insert it
	NewUserSecret := new(Schemas.UserSecret)
	NewUserSecret.ID = ""
	NewUserSecret.Secret = secret
	NewUserSecret.UserId = createdUser.ID
	NewUserSecret.Created = now
	NewUserSecret.Updated = now
	_, insertionError = UserSecretCollection.InsertOne(ctx.Context(), NewUserSecret)
	if insertionError != nil {
		return utilities.Response(utilities.ResponseParams{
			Ctx:    ctx,
			Info:   configuration.ResponseMessages.InternalServerError,
			Status: fiber.StatusInternalServerError,
		})
	}

	return utilities.Response(utilities.ResponseParams{
		Ctx: ctx,
	})
}
