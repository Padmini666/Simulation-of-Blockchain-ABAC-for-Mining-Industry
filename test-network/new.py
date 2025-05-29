import json
import subprocess
from datetime import datetime
import os
import re

# Load data
with open("generated_users.json") as f:
    users = json.load(f)

with open("generated_policies.json") as f:
    policies = json.load(f)

with open("mining_licences_sample.json") as f:
    resources = json.load(f)

FABRIC_CFG_PATH = "/Users/padmini/Monash/Masters/Semester4/Thesis/fabric-samples/test-network/compose/docker/peercfg"
log_csv = open("simulation_results.csv", "w")
log_csv.write("user_id,resource,status,latency_ms,timestamp\n")

def run_command(description, command, capture_result=False):
    try:
        env = os.environ.copy()
        env["FABRIC_CFG_PATH"] = FABRIC_CFG_PATH
        
        result = subprocess.run(command, shell=True, text=True, capture_output=True, env=env)
        
        if capture_result:
            match = re.search(r'AccessLog: \{.*?Requester:(.*?) Resource:(.*?) Status:(.*?) LatencyMs:(\d+) Timestamp:(.*?)\}', result.stdout)
            if match:
                user_id, resource, status, latency_ms, timestamp = match.groups()
                log_csv.write(f"{user_id.strip()},{resource.strip()},{status.strip()},{latency_ms.strip()},{timestamp.strip()}\n")
    except Exception as e:
        print(f"Error: {str(e)}")

# Step 1: Register Users
for user in users:
    user_id = user["user_id"]
    attrs = json.dumps(user["attributes"]).replace('"', '\\"')  # escape for shell
    cmd = f"""peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls \
--cafile "$PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem" \
-C mychannel -n simple \
--peerAddresses localhost:7051 \
--tlsRootCertFiles "$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem" \
--peerAddresses localhost:9051 \
--tlsRootCertFiles "$PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem" \
-c '{{"function":"RegisterUser","Args":["{user_id}", "{attrs}"]}}'"""
    run_command(f"Registering {user_id}...", cmd)

# Step 2: Create Policies
for resource in resources:
    res_id = resource["TNO"]
    if res_id in policies:
        conds = json.dumps(policies[res_id]).replace('"', '\\"')
        cmd = f"""peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com \
--tls \
--cafile "$PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem" \
-C mychannel -n simple \
--peerAddresses localhost:7051 \
--tlsRootCertFiles "$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem" \
--peerAddresses localhost:9051 \
--tlsRootCertFiles "$PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem" \
-c '{{"function":"CreatePolicy","Args":["{res_id}", "{conds}"]}}'"""
        run_command(f"Creating policy for {res_id}...", cmd)

# Step 3: Evaluate Access
for user in users:
    user_id = user["user_id"]
    for resource in resources:
        res_id = resource["TNO"]
        cmd = f"""peer chaincode query -C mychannel -n simple \
-c '{{"function":"EvaluateAccess","Args":["{user_id}", "{res_id}"]}}'"""
        run_command(f"Evaluating access for {user_id} to {res_id}", cmd, capture_result=True)

log_csv.close()
print("✅ CSV simulation log created at simulation_results.csv")

