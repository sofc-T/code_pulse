package service

import (
	"context"
	"fmt"
	"log"

	models "github.com/sofc-t/code_pulse/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Documentrepo interface {
	GetAllDocuments(email models.Email) ([]*models.Document, error)
	SearchDocuments(email string, searchQuery string) ([]*models.Document, error)
	CreateDocument(author string, title string, body interface{}, readAccess []string, writeAccess []string, language string) (string, error)
	UpdateDocument(documentID string, body models.DocumentData) error
	GetDocumentByID(documentID string) (*models.Document, error)
	UpdateTitle(documentID string, title string) (string, error)
	UpdateCollaborators(documentID string, collaborators models.Access) (models.Document, error)
	DeleteDocument(documentID string) (bool, error)
}

type documentrepo struct {
	collection *mongo.Collection // MongoDB collection
}

func NewDocumentrepo(client *mongo.Client, databaseName, collectionName string) Documentrepo {
	collection := client.Database(databaseName).Collection(collectionName)
	return &documentrepo{
		collection: collection,
	}
}

func (repo *documentrepo) GetAllDocuments(email models.Email) ([]*models.Document, error) {
	log.Printf("Getting all documents for email: %s", email.Email)
	filter := bson.M{"$or": []bson.M{{"author": email.Email}, {"readAccess": email.Email}, {"writeAccess": email.Email}}}
	cursor, err := repo.collection.Find(context.Background(), filter)
	if err != nil {
		log.Printf("Error finding documents: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var documents []*models.Document
	for cursor.Next(context.Background()) {
		var document models.Document
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Error decoding document: %v", err)
			return nil, err
		}
		documentmodels := &models.Document{
			ID:    document.ID,
			Title: document.Title,
		}

		documents = append(documents, documentmodels)
	}
	return documents, nil
}

func (repo *documentrepo) SearchDocuments(email string, searchQuery string) ([]*models.Document, error) {
	log.Printf("Searching documents for email: %s with query: %s", email, searchQuery)
	filter := bson.M{
		"$or": []bson.M{
			{"author": email},
			{"readAccess": email},
			{"writeAccess": email},
		},
		"title": bson.M{
			"$regex":   searchQuery,
			"$options": "i",
		},
	}
	cursor, err := repo.collection.Find(context.Background(), filter)
	if err != nil {
		log.Printf("Error finding documents: %v", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var documents []*models.Document
	for cursor.Next(context.Background()) {
		var document models.Document
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Error decoding document: %v", err)
			return nil, err
		}
		documentmodels := &models.Document{
			ID:    document.ID,
			Title: document.Title,
		}

		documents = append(documents, documentmodels)
	}
	return documents, nil
}

func (repo *documentrepo) CreateDocument(author string, title string, body interface{}, readAccess []string, writeAccess []string, language string) (string, error) {
	log.Printf("Creating document for author: %s with title: %s", author, title)
	newDocument := bson.D{
		{Key: "author", Value: author},
		{Key: "readAccess", Value: readAccess},
		{Key: "writeAccess", Value: writeAccess},
		{Key: "title", Value: title},
		{Key: "body", Value: body},
		{Key: "language", Value: language},
	}

	document, err := repo.collection.InsertOne(context.Background(), newDocument)
	if err != nil {
		log.Printf("Error inserting document: %v", err)
		return "", err
	}

	insertedID, ok := document.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Printf("Failed to convert InsertedID to string")
		return "", fmt.Errorf("failed to convert InsertedID to string")
	}

	return insertedID.Hex(), nil
}

func (repo *documentrepo) UpdateDocument(documentID string, incomingData models.DocumentData) error {
	log.Printf("Updating document with ID: %s", documentID)
	objectID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		log.Printf("Error converting documentID to ObjectID: %v", err)
		return err
	}

	update := bson.M{"$set": bson.M{"data.ops": incomingData.Ops}}
	filter := bson.M{"_id": objectID}
	_, err = repo.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("Error updating document: %v", err)
	}
	return err
}

func (repo *documentrepo) GetDocumentByID(documentID string) (*models.Document, error) {
	log.Printf("Getting document by ID: %s", documentID)
	var document models.Document
	objectID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		log.Printf("Error converting documentID to ObjectID: %v", err)
		return nil, err
	}

	filter := bson.M{"_id": objectID}
	err = repo.collection.FindOne(context.Background(), filter).Decode(&document)
	if err != nil {
		log.Printf("Error finding document: %v", err)
		return nil, err
	}

	return &document, nil
}

func (repo *documentrepo) UpdateTitle(documentID string, title string) (string, error) {
	log.Printf("Updating title of document with ID: %s to: %s", documentID, title)
	objectID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		log.Printf("Error converting documentID to ObjectID: %v", err)
		return "", err
	}
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"title": title}}
	_, err = repo.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("Error updating title: %v", err)
	}
	return "", err
}

func (repo *documentrepo) UpdateCollaborators(documentID string, collaborators models.Access) (models.Document, error) {
	log.Printf("Updating collaborators of document with ID: %s", documentID)
	objectID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		log.Printf("Error converting documentID to ObjectID: %v", err)
		return models.Document{}, err
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"readAccess": collaborators.ReadAccess, "writeAccess": collaborators.WriteAccess}}

	// Perform the update operation
	_, err = repo.collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Printf("Error updating collaborators: %v", err)
		return models.Document{}, err
	}

	// Retrieve the updated document
	var updatedDocument models.Document
	err = repo.collection.FindOne(context.Background(), filter).Decode(&updatedDocument)
	if err != nil {
		log.Printf("Error finding updated document: %v", err)
		return models.Document{}, err
	}
	return updatedDocument, nil
}

func (repo *documentrepo) DeleteDocument(documentID string) (bool, error) {
	log.Printf("Deleting document with ID: %s", documentID)
	objectID, err := primitive.ObjectIDFromHex(documentID)
	if err != nil {
		log.Printf("Error converting documentID to ObjectID: %v", err)
		return false, err
	}

	filter := bson.M{"_id": objectID}
	result, err := repo.collection.DeleteOne(context.Background(), filter)
	if err != nil {
		log.Printf("Error deleting document: %v", err)
		return false, err
	}

	return result.DeletedCount > 0, nil
}
