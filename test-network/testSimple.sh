#!/bin/bash

source ./scripts/envVar.sh

# Set value
setGlobals 1
echo "📝 Setting value (foo -> bar)..."
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com \
--tls --cafile "$PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem" \
-C mychannel -n simple \
--peerAddresses localhost:7051 \
--tlsRootCertFiles "$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem" \
-c '{"function":"Set","Args":["foo","bar"]}'

sleep 3

# Query same peer
setGlobals 1
echo "🔍 Getting value for key 'foo'..."
peer chaincode query -C mychannel -n simple -c '{"function":"Get","Args":["foo"]}'

