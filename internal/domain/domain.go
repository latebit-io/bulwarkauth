package domain

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionName = "domains"
)

type DomainService interface {
	Verify(ctx context.Context, domain string) error
	GenerateVerification(ctx context.Context, domain, salt string) (*DomainVerification, error)
	GetAll(ctx context.Context) ([]DomainVerification, error)
}

type DomainVerification struct {
	Domain      string    `bson:"domain"`
	Salt        string    `bson:"salt"`
	DNSRecord   string    `bson:"dnsRecord"`
	VerifyToken string    `bson:"verifyToken"`
	Verified    bool      `bson:"verified"`
	CreatedAt   time.Time `bson:"createdAt"`
	ModifiedAt  time.Time `bson:"modifiedAt"`
}

type DomainRepository interface {
	Create(ctx context.Context, domainVerification *DomainVerification) error
	Read(ctx context.Context, domain string) (*DomainVerification, error)
	ReadAll(ctx context.Context) ([]DomainVerification, error)
	Update(ctx context.Context, domainVerification *DomainVerification) error
}

type DefaultDomainService struct {
	domainRepository DomainRepository
	companyId        string
}

func (d DefaultDomainService) GetAll(ctx context.Context) ([]DomainVerification, error) {
	return d.domainRepository.ReadAll(ctx)
}

func (d DefaultDomainService) Verify(ctx context.Context, domain string) error {
	records, err := net.LookupTXT(domain)
	if err != nil {
		return err
	}

	for _, record := range records {
		if !strings.Contains(record, "bulwark-verify=") {
			continue
		}

		bulwarkKey := strings.Split(record, "bulwark-verify=")
		if len(bulwarkKey) != 2 {
			return VerificationError{
				Value: domain,
			}
		}
		record = strings.TrimSpace(bulwarkKey[1])

		domains, err := d.GetAll(ctx)

		if err != nil {
			return err
		}

		for _, dv := range domains {
			compare, err := d.generateToken(ctx, dv.Salt, dv.Domain, d.companyId)
			if err != nil {
				return err
			}
			if *compare == record {
				dv.Verified = true
				err = d.domainRepository.Update(ctx, &dv)
				return nil
			}
		}
	}

	return VerificationError{
		Value: domain,
	}
}

func (d DefaultDomainService) GenerateVerification(ctx context.Context, domain, salt string) (*DomainVerification, error) {
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	//salt := uuid.New().String()
	token, err := d.generateToken(ctx, salt, domain, d.companyId)
	if err != nil {
		return nil, err
	}

	dv := &DomainVerification{
		Domain:      domain,
		Salt:        salt,
		DNSRecord:   fmt.Sprintf("bulwark-verify=%s", *token),
		VerifyToken: *token,
		Verified:    false,
		CreatedAt:   time.Now(),
		ModifiedAt:  time.Now(),
	}

	err = d.domainRepository.Create(ctx, dv)
	if err != nil {
		return nil, err
	}

	return dv, nil
}

func (d DefaultDomainService) generateToken(ctx context.Context, salt, domain, companyId string) (*string, error) {
	h := hmac.New(sha256.New, []byte(salt))
	_, err := h.Write([]byte(domain + companyId))
	if err != nil {
		return nil, err
	}

	result := h.Sum(nil)
	token := base64.StdEncoding.EncodeToString(result)
	return &token, nil
}

func NewDefaultDomainService(domainRepository DomainRepository, companyId string) *DefaultDomainService {
	return &DefaultDomainService{
		domainRepository: domainRepository,
		companyId:        companyId,
	}
}

type DefaultDomainRepository struct {
	db *mongo.Database
}

func NewDefaultDomainRepository(db *mongo.Database) *DefaultDomainRepository {
	collection := db.Collection(collectionName)
	_, err := collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "domain", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Fatal(err)
	}
	return &DefaultDomainRepository{db}
}

func (d *DefaultDomainRepository) Create(ctx context.Context, domainVerification *DomainVerification) error {
	collection := d.db.Collection(collectionName)
	_, err := collection.InsertOne(ctx, domainVerification)
	if err != nil && mongo.IsDuplicateKeyError(err) {
		return DuplicateVerificationError{
			Value: domainVerification.Domain,
		}
	}
	return nil
}

func (d *DefaultDomainRepository) Read(ctx context.Context, domain string) (*DomainVerification, error) {
	collection := d.db.Collection(collectionName)
	var domainVerification DomainVerification
	err := collection.FindOne(ctx, bson.M{"domain": domain}).Decode(&domainVerification)
	if err != nil {
		return nil, err
	}
	return &domainVerification, nil
}

func (d *DefaultDomainRepository) ReadAll(ctx context.Context) ([]DomainVerification, error) {
	collection := d.db.Collection(collectionName)
	cursor, err := collection.Find(ctx, bson.M{})

	if err != nil {
		return nil, err
	}

	defer func() {
		err := cursor.Close(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Marshal results into a slice
	var results []DomainVerification
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (d *DefaultDomainRepository) Update(ctx context.Context, domainVerification *DomainVerification) error {
	collection := d.db.Collection(collectionName)
	_, err := collection.UpdateOne(ctx, bson.M{"domain": domainVerification.Domain}, bson.M{"$set": bson.M{
		"dnsRecord":   domainVerification.DNSRecord,
		"verifyToken": domainVerification.VerifyToken,
		"verified":    domainVerification.Verified,
		"createdAt":   domainVerification.CreatedAt,
	}})
	if err != nil {
		return err
	}
	return nil
}

type DomainCredentials struct {
	Domain    string    `bson:"domain"`
	ClientID  string    `bson:"client_id"`
	ClientKey string    `bson:"client_key"`
	CreatedAt time.Time `bson:"created_at"`
}

//// Middleware to verify requests
//func AuthenticateRequest(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		clientID := r.Header.Get("X-Client-ID")
//		signature := r.Header.Get("X-Signature")
//
//		// Verify the client credentials and signature
//		if !verifyClientCredentials(clientID, signature, r) {
//			http.Error(w, "Unauthorized", http.StatusUnauthorized)
//			return
//		}
//
//		next.ServeHTTP(w, r)
//	})
//}

//func verifyClientCredentials(clientid string, signature string, r *http.Request) bool {
//
//}
//
//func signRequest(clientKey string, payload []byte) string {
//	h := hmac.New(sha256.New, []byte(clientKey))
//	h.Write(payload)
//	return base64.StdEncoding.EncodeToString(h.Sum(nil))
//}
