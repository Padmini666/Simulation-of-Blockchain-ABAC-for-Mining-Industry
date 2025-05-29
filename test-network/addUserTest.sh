#!/bin/bash

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/

# Set peer environment for Org1
source ./scripts/envVar.sh
setGlobals 1

# Step 1: Register user1 with attributes
echo "⏳ Registering user1 with attributes..."
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls --cafile "$PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem" \
-C mychannel -n abac \
--peerAddresses localhost:7051 \
--tlsRootCertFiles "$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem" \
-c '{"function":"RegisterUser","Args":["user1", "[\"finance\", \"manager\"]"]}'

sleep 3

# ✅ Reset peer context (MUST for query in scripts)
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ENABLED=true

# Step 2: Read the registered user directly
echo "🔍 Reading user1 attributes from ledger..."
peer chaincode query -C mychannel -n abac \
-c '{"function":"ReadUser","Args":["user1"]}'

