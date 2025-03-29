use serde::Deserialize;

pub fn guess_id_for_urn(
    urn: &str,
) -> Result<(String, String, String, Option<Vec<String>>), Box<dyn std::error::Error>> {
    let parts: Vec<&str> = urn.split("::").collect();
    let name = parts.last().unwrap();
    let resource_type = parts[parts.len() - 2];

    let (id, options) = match resource_type {
        "kubernetes:core/v1:Namespace" => kubectl_get("namespace", name)?,
        "kubernetes:core/v1:ConfigMap" => kubectl_get("configmap", name)?,
        "kubernetes:core/v1:Secret" => kubectl_get("secret", name)?,
        "kubernetes:core/v1:ServiceAccount" => kubectl_get("serviceaccount", name)?,

        "kubernetes:core/v1:Service" => kubectl_get("service", name)?,
        "kubernetes:apps/v1:Deployment" => kubectl_get("deployment", name)?,
        "kubernetes:rbac.authorization.k8s.io/v1:ClusterRole" => kubectl_get("clusterrole", name)?,
        "kubernetes:external-secrets.io/v1beta1:ExternalSecret" => {
            kubectl_get("externalsecret", name)?
        }
        "kubernetes:external-secrets.io/v1beta1:ClusterSecretStore" => {
            kubectl_get("clustersecretstore", name)?
        }
        "kubernetes:external-secrets.io/v1beta1:SecretStore" => kubectl_get("secretstore", name)?,
        "kubernetes:rbac.authorization.k8s.io/v1:ClusterRoleBinding" => {
            kubectl_get("clusterrolebinding", name)?
        }
        "kubernetes:cert-manager.io/v1:ClusterIssuer" => kubectl_get("clusterissuer", name)?,
        "kubernetes:cert-manager.io/v1:Issuer" => kubectl_get("issuer", name)?,
        "kubernetes:cert-manager.io/v1:Certificate" => kubectl_get("certificate", name)?,
        "kubernetes:metallb.io/v1beta1:IPAddressPool" => kubectl_get("ipaddresspool", name)?,
        "kubernetes:networking.k8s.io/v1:Ingress" => kubectl_get("ingress", name)?,
        "kubernetes:core/v1:PersistentVolumeClaim" => kubectl_get("persistentvolumeclaim", name)?,
        "kubernetes:helm.sh/v3:Release" => (helm_get(name)?, None),
        _ => ("".to_string(), None),
    };

    Ok((resource_type.to_string(), name.to_string(), id, options))
}

fn kubectl_list_resources(resource_type: &str) -> Result<Vec<String>, Box<dyn std::error::Error>> {
    let output = std::process::Command::new("kubectl")
        .arg("get")
        .arg(resource_type)
        .arg("--all-namespaces")
        .arg("--no-headers")
        .arg("-o")
        .arg("go-template={{range .items}}{{if .metadata.namespace}}{{.metadata.namespace}}/{{end}}{{.metadata.name}}{{\"\\n\"}}{{end}}")
        .output()?;

    let resource_list = String::from_utf8(output.stdout)?;
    let resources: Vec<&str> = resource_list.split("\n").collect();

    Ok(resources.iter().map(|r| r.to_string()).collect())
}

fn kubectl_get(
    resource_type: &str,
    name: &str,
) -> Result<(String, Option<Vec<String>>), Box<dyn std::error::Error>> {
    let resources = kubectl_list_resources(resource_type)?;

    for resource in &resources {
        if resource.split("/").last().unwrap().starts_with(name) {
            return Ok((resource.to_string(), None));
        }
    }

    Ok(("".to_string(), Some(resources)))
}

fn helm_get(name: &str) -> Result<String, Box<dyn std::error::Error>> {
    // helm list --all-namespaces -o json
    let output = std::process::Command::new("helm")
        .arg("list")
        .arg("--all-namespaces")
        .arg("-o")
        .arg("json")
        .output()?;

    let helm_releases: Vec<HelmRelease> = serde_json::from_slice(&output.stdout)?;

    for release in helm_releases {
        if release.name.starts_with(name) {
            return Ok(format!("{}/{}", release.namespace, release.name));
        }
    }

    Err("Resource not found".into())
}

#[derive(Debug, Deserialize)]
struct HelmRelease {
    name: String,
    namespace: String,
    revision: String,
    updated: String,
    status: String,
    chart: String,
    app_version: String,
}
