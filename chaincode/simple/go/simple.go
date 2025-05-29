package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// User defines the structure for storing user details and their attributes.
type User struct {
	ID         string   `json:"id"`
	Attributes []string `json:"attributes"`
}

// AccessPolicy defines a policy based on attribute conditions for accessing a resource.
type AccessPolicy struct {
	Resource   string   `json:"resource"`
	Conditions []string `json:"conditions"`
}

// AccessLog stores the results of access attempts, used for auditing.
type AccessLog struct {
	DocType   string `json:"docType"` // Enables CouchDB indexing
	Requester string `json:"requester"`
	Resource  string `json:"resource"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	Timestamp string `json:"timestamp"`
}

// SimpleContract defines the main contract logic.
type SimpleContract struct {
	contractapi.Contract
}

// RegisterUser stores a user and their attribute list on the ledger.
func (s *SimpleContract) RegisterUser(ctx contractapi.TransactionContextInterface, userID string, attributesJSON string) error {
	log.Printf("RegisterUser called for userID=%s", userID)

	var attrs []string
	if err := json.Unmarshal([]byte(attributesJSON), &attrs); err != nil {
		log.Printf("Failed to parse attributes for %s: %v", userID, err)
		return fmt.Errorf("invalid attribute format: %v", err)
	}

	user := User{ID: userID, Attributes: attrs}
	data, _ := json.Marshal(user)

	err := ctx.GetStub().PutState("USER_"+userID, data)
	if err != nil {
		log.Printf("Failed to store user %s: %v", userID, err)
		return err
	}
	log.Printf("User registered: %s", userID)
	return nil
}

// CreatePolicy stores access conditions for a specific resource.
func (s *SimpleContract) CreatePolicy(ctx contractapi.TransactionContextInterface, resource string, conditionsJSON string) error {
	log.Printf("CreatePolicy called for resource=%s", resource)

	var conds []string
	if err := json.Unmarshal([]byte(conditionsJSON), &conds); err != nil {
		log.Printf("Failed to parse conditions for %s: %v", resource, err)
		return fmt.Errorf("invalid policy conditions format: %v", err)
	}
	policy := AccessPolicy{Resource: resource, Conditions: conds}
	data, _ := json.Marshal(policy)

	err := ctx.GetStub().PutState("POLICY_"+resource, data)
	if err != nil {
		log.Printf("Failed to store policy for %s: %v", resource, err)
		return err
	}
	log.Printf("Policy created for: %s", resource)
	return nil
}

// EvaluateAccess checks if a user satisfies the policy conditions for a resource.
func (s *SimpleContract) EvaluateAccess(ctx contractapi.TransactionContextInterface, userID string, resource string) (string, error) {
	log.Printf("EvaluateAccess called: userID=%s, resource=%s", userID, resource)
	start := time.Now()

	userData, err := ctx.GetStub().GetState("USER_" + userID)
	if err != nil || userData == nil {
		log.Printf("User not found or fetch error for %s", userID)
		return "", fmt.Errorf("user not found")
	}
	var user User
	if err := json.Unmarshal(userData, &user); err != nil {
		return "", fmt.Errorf("invalid user data: %v", err)
	}
	policyData, err := ctx.GetStub().GetState("POLICY_" + resource)
	if err != nil || policyData == nil {
		log.Printf("Policy not found or fetch error for %s", resource)
		return "", fmt.Errorf("policy not found")
	}
	var policy AccessPolicy
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return "", fmt.Errorf("invalid policy data: %v", err)
	}
	// Check each policy condition against user's attributes
	status := "Granted"
	for _, cond := range policy.Conditions {
		matched := false
		for _, attr := range user.Attributes {
			if attr == cond {
				matched = true
				break
			}
		}
		if !matched {
			status = "Denied"
			break
		}
	}
	// Log the access result
	logEntry := AccessLog{
		DocType:   "accessLog",
		Requester: user.ID,
		Resource:  policy.Resource,
		Status:    status,
		LatencyMs: time.Since(start).Milliseconds(),
		Timestamp: time.Now().Format(time.RFC3339),
	}
	logBytes, _ := json.Marshal(logEntry)
	txID := ctx.GetStub().GetTxID()
	logKey := fmt.Sprintf("LOG_%s_%s", userID, txID)
	err = ctx.GetStub().PutState(logKey, logBytes)
	if err != nil {
		return "", fmt.Errorf("failed to store access log: %v", err)
	}
	log.Printf("Access evaluation completed: %s", status)
	return fmt.Sprintf("AccessLog: %+v", logEntry), nil
}

// GetAccessLogs returns all access attempts by a user from the ledger.
func (s *SimpleContract) GetAccessLogs(ctx contractapi.TransactionContextInterface, userID string) ([]*AccessLog, error) {
	log.Printf("Fetching logs for userID=%s", userID)
	query := fmt.Sprintf(`{"selector":{"docType":"accessLog","requester":"%s"}}`, userID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query access logs: %v", err)
	}
	defer resultsIterator.Close()
	var logs []*AccessLog
	for resultsIterator.HasNext() {
		queryResponse, _ := resultsIterator.Next()
		var entry AccessLog
		_ = json.Unmarshal(queryResponse.Value, &entry)
		logs = append(logs, &entry)
		log.Printf("Log entry: %+v", entry)
	}
	return logs, nil
}

// main launches the chaincode.
func main() {
	cc, err := contractapi.NewChaincode(new(SimpleContract))
	if err != nil {
		log.Panicf("Failed to create chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Panicf("Failed to start chaincode: %v", err)
	}
}
