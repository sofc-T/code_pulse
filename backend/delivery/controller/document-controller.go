package controller

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	service "github.com/sofc-t/code_pulse/infrastructure/doc"
	models "github.com/sofc-t/code_pulse/models"
)

type DocumentController interface {
	GetAllDocuments(ctx *gin.Context) ([]*models.Document, error)
	SearchDocuments(ctx *gin.Context) ([]*models.Document, error)
	CreateNewDocument(ctx *gin.Context)
	UpdateDocument(documentID string, body models.DocumentData) error
	GetOneDocument(ctx *gin.Context) (*models.Document, error)
	UpdateTitle(ctx *gin.Context) (string, string)
	UpdateCollaborators(ctx *gin.Context) models.Document
	DeleteDocument(ctx *gin.Context)
}

type documentController struct {
	documentrepo service.Documentrepo
}

func NewDocumentController(documentrepo service.Documentrepo) DocumentController {
	return &documentController{
		documentrepo: documentrepo,
	}
}

func (controller *documentController) GetAllDocuments(ctx *gin.Context) ([]*models.Document, error) {
	// Implement logic to fetch all documents from the MongoDB collection of a single user
	var email models.Email
	err := ctx.ShouldBind(&email)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return nil, err
	}
	documents, err := controller.documentrepo.GetAllDocuments(email)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return nil, err
	}

	return documents, nil
}

func (contrller *documentController) SearchDocuments(ctx *gin.Context) ([]*models.Document, error) {
	var searchQuery models.Search
	// Implement logic to search for documents in the MongoDB collection of a single user
	err := ctx.ShouldBind(&searchQuery)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return nil, err
	}
	documents, err := contrller.documentrepo.SearchDocuments(searchQuery.Email, searchQuery.SearchQuery)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search documents"})
		return nil, err
	}

	return documents, nil
}

func (controller *documentController) CreateNewDocument(ctx *gin.Context) {
	var document models.Document
	if err := ctx.ShouldBindJSON(&document); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if document.Author == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Author is required"})
		return
	}
	if document.ReadAccess == nil {
		document.ReadAccess = []string{document.Author}
	}
	if document.WriteAccess == nil {
		document.WriteAccess = []string{document.Author}
	}

	documentID, err := controller.documentrepo.CreateDocument(document.Author, document.Title, document.Data, document.ReadAccess, document.WriteAccess)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "Document created successfully", "document_id": documentID})
}

func (controller *documentController) UpdateDocument(documentID string, body models.DocumentData) error {
	// Implement logic to update a document in the MongoDB collection of a single user
	err := controller.documentrepo.UpdateDocument(documentID, body)
	if err != nil {
		return err
	}
	return nil
}

func (controller *documentController) GetOneDocument(ctx *gin.Context) (*models.Document, error) {
	fmt.Println("GetOneDocument is called")
	documentID := ctx.Param("id")
	document, err := controller.documentrepo.GetDocumentByID(documentID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return nil, err
	}

	return document, nil
}

func (controller *documentController) UpdateTitle(ctx *gin.Context) (string, string) {
	var document models.Title
	if err := ctx.ShouldBindJSON(&document); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return "", ""
	}
	if document.ID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return "", ""
	}
	if document.Title == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
		return "", ""
	}
	_, err := controller.documentrepo.UpdateTitle(document.ID, document.Title)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document title"})
		return "", ""
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Document title updated successfully"})
	return document.Title, document.ID
}

func (controller *documentController) UpdateCollaborators(ctx *gin.Context) models.Document {
	var access models.Access
	if err := ctx.ShouldBindJSON(&access); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return models.Document{}
	}
	if len(access.ReadAccess) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Access is required"})
		return models.Document{}
	}
	if len(access.WriteAccess) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Access is required"})
		return models.Document{}
	}
	document, err := controller.documentrepo.UpdateCollaborators(access.ID, access)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document access"})
		return models.Document{}
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Document access updated successfully"})
	return document
}

func (controller *documentController) DeleteDocument(ctx *gin.Context) {
	documentID := ctx.Param("id")

	if documentID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}
	s, err := controller.documentrepo.DeleteDocument(documentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}
	if !s {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}
