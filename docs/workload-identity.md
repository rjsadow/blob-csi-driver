# How to Use workload identity with Blob CSI driver

## Prerequisites

This document is mainly refer to [Azure AD Workload Identity Quick Start](https://azure.github.io/azure-workload-identity/docs/quick-start.html). Please Complete the [Installation guide](https://azure.github.io/azure-workload-identity/docs/installation.html) before the following steps.

After you finish the Installation guide, you should have already:

* installed the mutating admission webhook
* obtained your cluster’s OIDC issuer URL

## 1. Export environment variables

```shell
export CLUSTER_NAME="<your cluster name>"
export CLUSTER_RESOURCE_GROUP="<cluster resource group name>"
export LOCATION="<location>"
export OIDC_ISSUER="<your cluster’s OIDC issuer URL>"

# [OPTIONAL] resource group where Blob storage account reside
export AZURE_BLOB_RESOURCE_GROUP="<resource group where Blob storage account reside>"

# environment variables for the AAD application
# [OPTIONAL] Only set this if you're using a Azure AD Application as part of this tutorial
export APPLICATION_NAME="<your application name>"

# environment variables for the user-assigned managed identity
# [OPTIONAL] Only set this if you're using a user-assigned managed identity as part of this tutorial
export USER_ASSIGNED_IDENTITY_NAME="<your user-assigned managed identity name>"
export IDENTITY_RESOURCE_GROUP="<resource group where your user-assigned managed identity reside>"

# Blob CSI Driver Service Account and namespace
export SA_LIST=( "csi-blob-controller-sa" "csi-blob-node-sa" )
export NAMESPACE="kube-system"
```

## 2. Create Blob resource group

If you are using AKS, you can get the resource group where Blob storage class reside by running:

```shell
export AZURE_BLOB_RESOURCE_GROUP="$(az aks show --name $CLUSTER_NAME --resource-group $CLUSTER_RESOURCE_GROUP --query "nodeResourceGroup" -o tsv)"
```

You can also create resource group by yourself, but you must [specify the resource group](https://github.com/cvvz/blob-csi-driver/blob/workload_identity/docs/driver-parameters.md) in the storage class while using Blob CSI driver:

```shell
az group create -n $AZURE_BLOB_RESOURCE_GROUP -l $LOCATION
```

## 3. Create an AAD application or user-assigned managed identity and grant required permissions 

```shell
# create an AAD application if using Azure AD Application for this tutorial
az ad sp create-for-rbac --name "${APPLICATION_NAME}"
```

```shell
# create a user-assigned managed identity if using user-assigned managed identity for this tutorial
az group create -n ${IDENTITY_RESOURCE_GROUP} -l $LOCATION
az identity create --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${IDENTITY_RESOURCE_GROUP}"
```

Grant required permission to the AAD application or user-assigned managed identity, for simplicity, we just assign Contributor role to the resource group where Blob storage class reside:

If using Azure AD Application:

```shell
export APPLICATION_CLIENT_ID="$(az ad sp list --display-name "${APPLICATION_NAME}" --query '[0].appId' -otsv)"
export AZURE_BLOB_RESOURCE_GROUP_ID="$(az group show -n $AZURE_BLOB_RESOURCE_GROUP --query 'id' -otsv)"
az role assignment create --assignee $APPLICATION_CLIENT_ID --role Contributor --scope $AZURE_BLOB_RESOURCE_GROUP_ID
```

if using user-assigned managed identity:

```shell
export USER_ASSIGNED_IDENTITY_OBJECT_ID="$(az identity show --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${IDENTITY_RESOURCE_GROUP}" --query 'principalId' -otsv)"
export AZURE_BLOB_RESOURCE_GROUP_ID="$(az group show -n $AZURE_BLOB_RESOURCE_GROUP --query 'id' -otsv)"
az role assignment create --assignee $USER_ASSIGNED_IDENTITY_OBJECT_ID --role Contributor --scope $AZURE_BLOB_RESOURCE_GROUP_ID
```

## 4. Establish federated identity credential between the identity and the Blob service account issuer & subject

If using Azure AD Application:

```shell
# Get the object ID of the AAD application
export APPLICATION_OBJECT_ID="$(az ad app show --id ${APPLICATION_CLIENT_ID} --query id -otsv)"

# Add the federated identity credential:
for SERVICE_ACCOUNT_NAME in "${SA_LIST[@]}"
do
cat <<EOF > params.json
{
  "name": "${SERVICE_ACCOUNT_NAME}",
  "issuer": "${OIDC_ISSUER}",
  "subject": "system:serviceaccount:${NAMESPACE}:${SERVICE_ACCOUNT_NAME}",
  "description": "Kubernetes service account federated credential",
  "audiences": [
    "api://AzureADTokenExchange"
  ]
}
EOF
az ad app federated-credential create --id ${APPLICATION_OBJECT_ID} --parameters @params.json
done
```

If using user-assigned managed identity:

```shell
for SERVICE_ACCOUNT_NAME in "${SA_LIST[@]}"
do
az identity federated-credential create \
--name "${SERVICE_ACCOUNT_NAME}" \
--identity-name "${USER_ASSIGNED_IDENTITY_NAME}" \
--resource-group "${IDENTITY_RESOURCE_GROUP}" \
--issuer "${OIDC_ISSUER}" \
--subject system:serviceaccount:"${NAMESPACE}":"${SERVICE_ACCOUNT_NAME}"
done
```

## 5. Deploy Blob CSI Driver

Deploy storageclass:

```shell
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/blob-csi-driver/master/deploy/example/storageclass-blobfuse.yaml
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/blob-csi-driver/master/deploy/example/storageclass-blob-nfs.yaml
```

Deploy Blob CSI Driver

If using Azure AD Application:

```shell
export CLIENT_ID="$(az ad sp list --display-name "${APPLICATION_NAME}" --query '[0].appId' -otsv)"
export TENANT_ID="$(az ad sp list --display-name "${APPLICATION_NAME}" --query '[0].appOwnerOrganizationId' -otsv)"
helm install blob-csi-driver charts/latest/blob-csi-driver \
--namespace $NAMESPACE \
--set workloadIdentity.clientID=$CLIENT_ID \
--set workloadIdentity.tenantID=$TENANT_ID
```

If using user-assigned managed identity:

```shell
export CLIENT_ID="$(az identity show --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${IDENTITY_RESOURCE_GROUP}" --query 'clientId' -otsv)"
export TENANT_ID="$(az identity show --name "${USER_ASSIGNED_IDENTITY_NAME}" --resource-group "${IDENTITY_RESOURCE_GROUP}" --query 'tenantId' -otsv)"
helm install blob-csi-driver charts/latest/blob-csi-driver \
--namespace $NAMESPACE \
--set workloadIdentity.clientID=$CLIENT_ID \
--set workloadIdentity.tenantID=$TENANT_ID
```

## 6. Deploy application using Blob CSI driver

```shell
kubectl create -f https://raw.githubusercontent.com/kubernetes-sigs/blob-csi-driver/master/deploy/example/nfs/statefulset.yaml
kubectl create -f  https://raw.githubusercontent.com/kubernetes-sigs/blob-csi-driver/master/deploy/example/deployment.yaml
```

Please make sure all the Pods are running.
