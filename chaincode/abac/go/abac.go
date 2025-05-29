package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type User struct {
	ID         string   `json:"id"`
	Attributes []string `json:"attributes"`
}

type AccessPolicy struct {
	Resource   string   `json:"resource"`
	Conditions []string `json:"conditions"`
}

type AccessRequest struct {
	UserID   string `json:"userID"`
	Resource string `json:"resource"`
}

type AccessLog struct {
	Requester string `json:"requester"`
	Resource  string `json:"resource"`
	Status    string `json:"status"`
}

func contains(list []string, condition string) bool {
	for _, attr := range list {
		if attr == condition {
			return true
		}
	}
	return false
}

func (s *SmartContract) RegisterUser(ctx contractapi.TransactionContextInterface, userID string, attributesJSON string) error {
	fmt.Printf("🛠️ RegisterUser invoked with ID: %s and attributes: %s\n", userID, attributesJSON)

	var attrs []string
	err := json.Unmarshal([]byte(attributesJSON), &attrs)
	if err != nil {
		return fmt.Errorf("❌ Invalid attributes format: %v", err)
	}

	user := User{ID: userID, Attributes: attrs}
	userBytes, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("❌ Failed to marshal user: %v", err)
	}

	key := "USER_" + userID
	fmt.Printf("📦 Writing to ledger with key: %s\n", key)
	return ctx.GetStub().PutState(key, userBytes)
}

func (s *SmartContract) DefinePolicy(ctx contractapi.TransactionContextInterface, resource string, policyJSON string) error {
	var policy AccessPolicy
	err := json.Unmarshal([]byte(policyJSON), &policy)
	if err != nil {
		return fmt.Errorf("Invalid policy format: %v", err)
	}
	policy.Resource = resource
	policyBytes, _ := json.Marshal(policy)
	return ctx.GetStub().PutState("POLICY_"+resource, policyBytes)
}

func (s *SmartContract) RequestAccess(ctx contractapi.TransactionContextInterface, requestJSON string) (string, error) {
	var request AccessRequest
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return "", fmt.Errorf("Invalid request format: %v", err)
	}
	userBytes, err := ctx.GetStub().GetState("USER_" + request.UserID)
	if err != nil || userBytes == nil {
		return "Denied", fmt.Errorf("User not found")
	}
	var user User
	json.Unmarshal(userBytes, &user)
	policyBytes, err := ctx.GetStub().GetState("POLICY_" + request.Resource)
	if err != nil || policyBytes == nil {
		return "Denied", fmt.Errorf("No policy found for resource")
	}
	var policy AccessPolicy
	json.Unmarshal(policyBytes, &policy)
	for _, cond := range policy.Conditions {
		if !contains(user.Attributes, cond) {
			s.logAccess(ctx, user.ID, request.Resource, "Denied")
			return "Denied", nil
		}
	}
	s.logAccess(ctx, user.ID, request.Resource, "Granted")
	return "Access Granted", nil
}

func (s *SmartContract) logAccess(ctx contractapi.TransactionContextInterface, userID, resource, status string) {
	log := AccessLog{Requester: userID, Resource: resource, Status: status}
	logBytes, _ := json.Marshal(log)
	logKey := "LOG_" + userID + "_" + resource
	ctx.GetStub().PutState(logKey, logBytes)
}

func (s *SmartContract) ReadUser(ctx contractapi.TransactionContextInterface, userID string) (*User, error) {
	userBytes, err := ctx.GetStub().GetState("USER_" + userID)
	if err != nil || userBytes == nil {
		return nil, fmt.Errorf("User not found")
	}
	var user User
	_ = json.Unmarshal(userBytes, &user)
	return &user, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		panic(fmt.Sprintf("Error creating ABAC chaincode: %v", err))
	}
	if err := chaincode.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start ABAC chaincode: %v", err))
	}
}

