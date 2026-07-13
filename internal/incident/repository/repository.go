package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	incidentdomain "github.com/lab2/rest-api/internal/incident/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrVersionConflict = errors.New("incident version conflict")

type Repository interface {
	Create(ctx context.Context, incident *incidentdomain.Incident, event *incidentdomain.TimelineEvent) error
	Get(ctx context.Context, id uuid.UUID) (*incidentdomain.Incident, error)
	List(ctx context.Context, filter incidentdomain.Filter) (*incidentdomain.Page, error)
	Update(ctx context.Context, incident *incidentdomain.Incident, expectedVersion int64) error
	AppendTimeline(ctx context.Context, event *incidentdomain.TimelineEvent) error
	Timeline(ctx context.Context, incidentID uuid.UUID) ([]*incidentdomain.TimelineEvent, error)
}

type MongoRepository struct {
	incidents *mongo.Collection
	timeline  *mongo.Collection
}

func NewMongoRepository(incidents, timeline *mongo.Collection) *MongoRepository {
	return &MongoRepository{incidents: incidents, timeline: timeline}
}

func (r *MongoRepository) Create(ctx context.Context, incident *incidentdomain.Incident, event *incidentdomain.TimelineEvent) error {
	if incident.ID == uuid.Nil {
		incident.ID = uuid.New()
	}
	now := time.Now().UTC()
	incident.CreatedAt, incident.UpdatedAt, incident.Version = now, now, 1
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.IncidentID, event.CreatedAt = incident.ID, now
	if _, err := r.incidents.InsertOne(ctx, incident); err != nil {
		return err
	}
	if _, err := r.timeline.InsertOne(ctx, event); err != nil {
		_, _ = r.incidents.DeleteOne(ctx, bson.M{"_id": incident.ID})
		return err
	}
	return nil
}

func (r *MongoRepository) Get(ctx context.Context, id uuid.UUID) (*incidentdomain.Incident, error) {
	var incident incidentdomain.Incident
	err := r.incidents.FindOne(ctx, bson.M{"_id": id}).Decode(&incident)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &incident, err
}

func (r *MongoRepository) List(ctx context.Context, filter incidentdomain.Filter) (*incidentdomain.Page, error) {
	query := bson.M{}
	if filter.Status.Valid() {
		query["status"] = filter.Status
	}
	if filter.Severity.Valid() {
		query["severity"] = filter.Severity
	}
	if filter.Service != "" {
		query["service"] = filter.Service
	}
	if filter.AssigneeID != nil {
		query["assignee_id"] = *filter.AssigneeID
	}
	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	total, err := r.incidents.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	cursor, err := r.incidents.Find(ctx, query, options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetSkip((page-1)*limit).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	data := make([]*incidentdomain.Incident, 0)
	if err := cursor.All(ctx, &data); err != nil {
		return nil, err
	}
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return &incidentdomain.Page{Data: data, Page: page, Limit: limit, Total: total, TotalPages: totalPages}, nil
}

func (r *MongoRepository) Update(ctx context.Context, incident *incidentdomain.Incident, expectedVersion int64) error {
	now := time.Now().UTC()
	update := bson.M{"$set": bson.M{
		"title": incident.Title, "description": incident.Description, "service": incident.Service,
		"severity": incident.Severity, "status": incident.Status, "assignee_id": incident.AssigneeID,
		"updated_at": now,
	}, "$inc": bson.M{"version": 1}}
	result, err := r.incidents.UpdateOne(ctx, bson.M{"_id": incident.ID, "version": expectedVersion}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		existing, getErr := r.Get(ctx, incident.ID)
		if getErr != nil {
			return getErr
		}
		if existing == nil {
			return mongo.ErrNoDocuments
		}
		return ErrVersionConflict
	}
	incident.Version = expectedVersion + 1
	incident.UpdatedAt = now
	return nil
}

func (r *MongoRepository) AppendTimeline(ctx context.Context, event *incidentdomain.TimelineEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	_, err := r.timeline.InsertOne(ctx, event)
	return err
}

func (r *MongoRepository) Timeline(ctx context.Context, incidentID uuid.UUID) ([]*incidentdomain.TimelineEvent, error) {
	cursor, err := r.timeline.Find(ctx, bson.M{"incident_id": incidentID}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := make([]*incidentdomain.TimelineEvent, 0)
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}
