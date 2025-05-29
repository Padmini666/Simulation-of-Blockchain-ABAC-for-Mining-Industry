#!/bin/bash

# Set Fabric environment variables (Org1 as default)
export CORE_PEER_LOCALMSPID=Org1MSP
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

ORDERER_CA="$PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem"
CHANNEL_NAME="mychannel"
CHAINCODE_NAME="simple"

echo "🚀 Running full access control simulation..."

> simulation_results.log

# Users and attributes
declare -A users
users["user01"]='["CertifiedMiner","Region=AU","GoldSpecialist"]'
users["user02"]='["Licensed","Region=AU","GypsumExpert"]'
users["user03"]='["CertifiedMiner","Region=NZ","GoldSpecialist"]'
users["user04"]='["CertifiedMiner","Region=AU","KaolinExpert"]'
users["user05"]='["Intern","Region=AU","GoldSpecialist"]'
users["user06"]='["CertifiedMiner","Region=AU","SilverExpert"]'
users["user07"]='["Licensed","Region=AU","CoalExpert"]'
users["user08"]='["CertifiedMiner","Region=AU","PlatinumExpert"]'
users["user09"]='["CertifiedMiner","Region=AU"]'
users["user10"]='["Licensed","Region=AU","GoldSpecialist"]'

# Register users
for user in "${!users[@]}"; do
  ATTRS=${users[$user]}
  echo "👤 Registering $user..."
  peer chaincode invoke -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile $ORDERER_CA \
  -C $CHANNEL_NAME -n $CHAINCODE_NAME \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
  --peerAddresses localhost:9051 \
  --tlsRootCertFiles $PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem \
  -c "{\"function\":\"RegisterUser\",\"Args\":[\"$user\", \"$ATTRS\"]}"
done

# Policies
declare -A policies
policies["GoldValuationData"]='["CertifiedMiner","Region=AU","GoldSpecialist"]'
policies["GypsumMiningInfo"]='["Licensed","Region=AU","GypsumExpert"]'
policies["CoalEnergyData"]='["Licensed","Region=AU","CoalExpert"]'
policies["PreciousMetalsReport"]='["CertifiedMiner","Region=AU","PlatinumExpert","SilverExpert"]'
policies["KaolinReserveAccess"]='["CertifiedMiner","Region=AU","KaolinExpert"]'

# Create policies
for resource in "${!policies[@]}"; do
  CONDITIONS=${policies[$resource]}
  echo "📜 Creating policy for $resource..."
  peer chaincode invoke -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls --cafile $ORDERER_CA \
  -C $CHANNEL_NAME -n $CHAINCODE_NAME \
  --peerAddresses localhost:7051 \
  --tlsRootCertFiles $PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
  --peerAddresses localhost:9051 \
  --tlsRootCertFiles $PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem \
  -c "{\"function\":\"CreatePolicy\",\"Args\":[\"$resource\", \"$CONDITIONS\"]}"
done

# Evaluate access
for user in "${!users[@]}"; do
  for resource in "${!policies[@]}"; do
    echo "🔎 Evaluating access for $user to $resource"
    RESULT=$(peer chaincode query -C $CHANNEL_NAME -n $CHAINCODE_NAME \
      -c "{\"function\":\"EvaluateAccess\",\"Args\":[\"$user\", \"$resource\"]}")
    echo "$user,$resource,$RESULT" >> simulation_results.log
  done
done

echo "✅ Simulation completed. Results saved to simulation_results.log"

