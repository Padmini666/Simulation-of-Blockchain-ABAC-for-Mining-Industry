package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type User struct {
	ID         string   `json:"id"`
	Attributes []string `json:"attributes"`
}

type AccessPolicy struct {
	Resource   string   `json:"resource"`
	Conditions []string `json:"conditions"`
}

type AccessLog struct {
	DocType   string `json:"docType"` // Enables CouchDB indexing
	Requester string `json:"requester"`
	Resource  string `json:"resource"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	Timestamp string `json:"timestamp"`
}

type SimpleContract struct {
	contractapi.Contract
}

func (s *SimpleContract) RegisterUser(ctx contractapi.TransactionContextInterface, userID string, attributesJSON string) error {
	log.Printf("RegisterUser called for userID=%s", userID)

	var attrs []string
	if err := json.Unmarshal([]byte(attributesJSON), &attrs); err != nil {
		log.Printf("RegisterUser failed to parse attributes for userID=%s: %v", userID, err)
		return fmt.Errorf("invalid attribute format: %v", err)
	}

	user := User{ID: userID, Attributes: attrs}
	data, _ := json.Marshal(user)

	err := ctx.GetStub().PutState("USER_"+userID, data)
	if err != nil {
		log.Printf("RegisterUser failed to store user %s: %v", userID, err)
		return err
	}
	log.Printf("RegisterUser succeeded for userID=%s", userID)
	return nil
}

func (s *SimpleContract) CreatePolicy(ctx contractapi.TransactionContextInterface, resource string, conditionsJSON string) error {
	log.Printf("CreatePolicy called for resource=%s", resource)

	var conds []string
	if err := json.Unmarshal([]byte(conditionsJSON), &conds); err != nil {
		log.Printf("CreatePolicy failed to parse conditions for resource=%s: %v", resource, err)
		return fmt.Errorf("invalid policy conditions format: %v", err)
	}

	policy := AccessPolicy{Resource: resource, Conditions: conds}
	data, _ := json.Marshal(policy)

	err := ctx.GetStub().PutState("POLICY_"+resource, data)
	if err != nil {
		log.Printf("CreatePolicy failed to store policy for resource=%s: %v", resource, err)
		return err
	}
	log.Printf("CreatePolicy succeeded for resource=%s", resource)
	return nil
}

func (s *SimpleContract) EvaluateAccess(ctx contractapi.TransactionContextInterface, userID string, resource string) (string, error) {
	log.Printf("EvaluateAccess called: userID=%s, resource=%s", userID, resource)
	start := time.Now()

	userData, err := ctx.GetStub().GetState("USER_" + userID)
	if err != nil {
		log.Printf("EvaluateAccess failed to get user data: %v", err)
		return "", fmt.Errorf("failed to retrieve user: %v", err)
	}
	if userData == nil {
		log.Printf("EvaluateAccess: user %s not found", userID)
		return "", fmt.Errorf("user not found")
	}

	var user User
	if err := json.Unmarshal(userData, &user); err != nil {
		log.Printf("EvaluateAccess: user data invalid for %s: %v", userID, err)
		return "", fmt.Errorf("invalid user data: %v", err)
	}

	policyData, err := ctx.GetStub().GetState("POLICY_" + resource)
	if err != nil {
		log.Printf("EvaluateAccess failed to get policy data: %v", err)
		return "", fmt.Errorf("failed to retrieve policy: %v", err)
	}
	if policyData == nil {
		log.Printf("EvaluateAccess: policy %s not found", resource)
		return "", fmt.Errorf("policy not found")
	}

	var policy AccessPolicy
	if err := json.Unmarshal(policyData, &policy); err != nil {
		log.Printf("EvaluateAccess: policy data invalid for %s: %v", resource, err)
		return "", fmt.Errorf("invalid policy data: %v", err)
	}

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

	logEntry := AccessLog{
		DocType:   "accessLog", // Important for CouchDB queries
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
		log.Printf("EvaluateAccess failed to store log: %v", err)
		return "", fmt.Errorf("failed to store access log: %v", err)
	}

	log.Printf("EvaluateAccess result: %s", logEntry.Status)
	return fmt.Sprintf("AccessLog: %+v", logEntry), nil
}

func (s *SimpleContract) GetAccessLogs(ctx contractapi.TransactionContextInterface, userID string) ([]*AccessLog, error) {
	log.Printf("GetAccessLogs called for userID=%s", userID)

	queryString := fmt.Sprintf(`{"selector":{"docType":"accessLog","requester":"%s"}}`, userID)
	log.Printf("Running CouchDB rich query: %s", queryString)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return nil, fmt.Errorf("failed to run CouchDB query: %v", err)
	}
	defer resultsIterator.Close()

	var logs []*AccessLog
	for resultsIterator.HasNext() {
		queryResponse, _ := resultsIterator.Next()
		var entry AccessLog
		_ = json.Unmarshal(queryResponse.Value, &entry)
		logs = append(logs, &entry)
		log.Printf("Retrieved log entry: %+v", entry)
	}

	log.Printf("Total logs found for userID=%s: %d", userID, len(logs))
	return logs, nil
}

func main() {
	cc, err := contractapi.NewChaincode(new(SimpleContract))
	if err != nil {
		log.Panicf("Failed to create chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Panicf("Failed to start chaincode: %v", err)
	}
}

