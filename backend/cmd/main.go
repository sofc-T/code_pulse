package main

import (
	"context"
	
	"fmt"
	
	"net/http"
	"os"
	
	

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sofc-t/code_pulse/delivery/controller"
	core "github.com/sofc-t/code_pulse/delivery/core"
	"github.com/sofc-t/code_pulse/models"
	docservice "github.com/sofc-t/code_pulse/infrastructure/doc"
	userservice "github.com/sofc-t/code_pulse/infrastructure/user"
	
	"github.com/sofc-t/code_pulse/socket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)





func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	dbUrl := os.Getenv("DATABASE_URL")

	if dbUrl == "" {
		fmt.Println("DATABASE_URL not found in .env file")
		return
	}

	// MongoDB connection setup
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(dbUrl))
	if err != nil {
		panic(err)
	}
	defer mongoClient.Disconnect(context.Background())

	err = mongoClient.Ping(context.Background(), nil)
	if err != nil {
		fmt.Println("Failed to connect to MongoDB:", err)
		return
	} else {
		fmt.Println("Successfully connected to MongoDB!")
	}

	server := gin.New()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Authorization", "Content-Type"}

	server.Use(cors.New(config))

	server.Use(gin.Recovery(), gin.Logger())

	signupService := userservice.NewSignupService(mongoClient, "godoc", "users")
	signupController := controller.NewSignupController(signupService)

	loginService := userservice.NewLoginService(mongoClient, "godoc", "users")
	jwtService := core.NewJWTService()
	loginController := controller.NewLoginController(loginService, jwtService)

	// Route for chaecking the health of the server
	server.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "UP",
		})
	})

	// Routes for handling user authentication
	authRoutes := server.Group("/auth")
	{
		// Signup Endpoint: User creation
		authRoutes.POST("/signup", func(ctx *gin.Context) {
			message := signupController.Signup(ctx)
			if message != "" {
				ctx.JSON(http.StatusOK, gin.H{
					"message": message,
				})
			} else {
				ctx.JSON(http.StatusBadRequest, nil)
			}
		})

		// Login Endpoint: Authentication + Token creation
		authRoutes.POST("/login", func(ctx *gin.Context) {
			token := loginController.Login(ctx)
			if token != "" {
				ctx.JSON(http.StatusOK, gin.H{
					"token": token,
				})
			} else {
				ctx.JSON(http.StatusUnauthorized, gin.H{
					"message": "Invalid credentials",
				})
			}
		})
	}

	documentService := docservice.NewDocumentrepo(mongoClient, "godoc", "documents")
	documentController := controller.NewDocumentController(documentService)

	// Route for handling document operations
	documentRoutes := server.Group(("/documents"))
	// documentRoutes.Use(middlewares.AuthorizeJWT())
	{
		documentRoutes.GET("/handler", func(ctx *gin.Context) {
			documentID := ctx.Query("document_id")
			socket.HandleWebSocket(ctx, documentID)
		})

		// Route for getting all documents
		documentRoutes.POST("/getall", func(ctx *gin.Context) {
			// Fetching documents from MongoDB and responding with JSON
			documents, err := documentController.GetAllDocuments(ctx)
			if err != nil {
				return
			}
			ctx.JSON(http.StatusOK, gin.H{
				"documents": documents,
			})
		})

		// Route for seraching documents by title
		documentRoutes.POST("/search", func(ctx *gin.Context) {
			documents, err := documentController.SearchDocuments(ctx)
			if err != nil {
				fmt.Println("Error searching documents:", err)
				return
			}
			ctx.JSON(http.StatusOK, gin.H{
				"documents": documents,
			})
		})

		// Route for creating a new document
		documentRoutes.POST("/createnew", func(ctx *gin.Context) {
			// Creating a new document and storing it in MongoDB
			documentController.CreateNewDocument(ctx)
		})

		// Route for getting a specific document
		documentRoutes.POST("/getone/:id", func(ctx *gin.Context) {

			document, err := socket.InitializeDocumentCache(ctx, documentController)

			if err != nil {
				fmt.Println("Error getting document:", err)
				return
			}
			ctx.JSON(http.StatusOK, document)
		})

		documentRoutes.POST("/updatetitle", func(ctx *gin.Context) {
			// Updating the title of a document
			title, documentID := documentController.UpdateTitle(ctx)
			socket.UpdateDocumentTitleCacheAttribute(documentID, title)
		})

		documentRoutes.POST("/updatecollaborators", func(ctx *gin.Context) {
			// Adding a collaborator to a document
			// updating the database
			document := documentController.UpdateCollaborators(ctx)
			// updating the cache
			access := new(models.Access)
			access.ID = document.ID
			access.ReadAccess = document.ReadAccess
			access.WriteAccess = document.WriteAccess
			socket.UpdateDocumentCacheAttribute(document.ID, *access)
		})

		// Route for deleting a document
		documentRoutes.DELETE("/delete/:id", func(ctx *gin.Context) {
			
			documentController.DeleteDocument(ctx)
		})
	}

	// Start the periodic cache update
	socket.UpdateDatabaseWithCache(documentController)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server.Run("0.0.0.0:" + port)
}


