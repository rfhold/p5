package builtins

import (
	"context"
	"errors"
	"strings"

	"github.com/rfhold/p5/internal/plugins"
	"github.com/rfhold/p5/internal/plugins/proto"
	"github.com/rfhold/p5/pkg/plugin"
)

var (
	errGrafanaURLNotConfigured  = errors.New("grafana url not configured")
	errEscalationChainIDMissing = errors.New("escalation chain id not found")
	errIntegrationIDMissing     = errors.New("integration id not found")
	errScheduleIDMissing        = errors.New("schedule id not found")
	errTeamUIDMissing           = errors.New("team uid not found")
	errRuleGroupFolderMissing   = errors.New("rule group folder uid not found")
	errRuleGroupNameMissing     = errors.New("rule group name not found")
	errDashboardURLMissing      = errors.New("dashboard url not found in outputs")
)

const irmAppPath = "/a/grafana-irm-app"

func init() {
	plugins.RegisterBuiltin(&GrafanaPlugin{
		BuiltinPluginBase: plugins.NewBuiltinPluginBase("grafana"),
	})
}

// GrafanaPlugin provides resource opening capabilities for Grafana resources
// by generating URLs to the Grafana console.
type GrafanaPlugin struct {
	plugins.BuiltinPluginBase
}

// Authenticate returns a no-op success response.
// This plugin is primarily for resource opening, not auth.
func (p *GrafanaPlugin) Authenticate(ctx context.Context, req *proto.AuthenticateRequest) (*proto.AuthenticateResponse, error) {
	return plugins.SuccessResponse(nil, 0), nil
}

// GetSupportedOpenTypes returns regex patterns for Grafana resource types.
func (p *GrafanaPlugin) GetSupportedOpenTypes(ctx context.Context, req *plugin.SupportedOpenTypesRequest) (*plugin.SupportedOpenTypesResponse, error) {
	return plugin.SupportedOpenTypesPatterns(
		`^grafana:onCall/.*`,
		`^grafana:oss/team:Team$`,
		`^grafana:oss/dashboard:Dashboard$`,
		`^grafana:alerting/ruleGroup:RuleGroup$`,
		`^grafana:alerting/contactPoint:ContactPoint$`,
		`^grafana:alerting/muteTiming:MuteTiming$`,
		`^grafana:alerting/notificationPolicy:NotificationPolicy$`,
		`^grafana:cloud/accessPolicy:AccessPolicy$`,
		`^grafana:cloud/accessPolicyToken:AccessPolicyToken$`,
		`^grafana:cloud/stackServiceAccount:StackServiceAccount$`,
		`^grafana:cloud/stackServiceAccountToken:StackServiceAccountToken$`,
	), nil
}

// OpenResource returns a browser URL to open a Grafana resource.
func (p *GrafanaPlugin) OpenResource(ctx context.Context, req *plugin.OpenResourceRequest) (*plugin.OpenResourceResponse, error) {
	grafanaURL := req.ProviderInputs["url"]
	if grafanaURL == "" {
		grafanaURL = req.StackConfig["grafana:url"]
	}
	if grafanaURL == "" {
		grafanaURL = req.ProgramConfig["grafana:url"]
	}
	grafanaURL = strings.TrimSuffix(grafanaURL, "/")

	url, err := p.buildResourceURL(req, grafanaURL)
	if err != nil {
		return plugin.OpenError("%v", err), nil
	}
	if url == "" {
		return plugin.OpenNotSupported(), nil
	}

	return plugin.OpenBrowserResponse(url), nil
}

func (p *GrafanaPlugin) buildResourceURL(req *plugin.OpenResourceRequest, grafanaURL string) (string, error) {
	switch req.ResourceType {
	case "grafana:onCall/escalationChain:EscalationChain":
		return p.buildEscalationChainURL(req.Outputs, grafanaURL)
	case "grafana:onCall/escalation:Escalation":
		return p.buildEscalationURL(req.Inputs, grafanaURL)
	case "grafana:onCall/integration:Integration":
		return p.buildIntegrationURL(req.Outputs, grafanaURL)
	case "grafana:onCall/onCallShift:OnCallShift":
		return p.buildOnCallShiftURL(grafanaURL)
	case "grafana:onCall/route:Route":
		return p.buildRouteURL(req.Inputs, grafanaURL)
	case "grafana:onCall/schedule:Schedule":
		return p.buildScheduleURL(req.Outputs, grafanaURL)
	case "grafana:oss/team:Team":
		return p.buildTeamURL(req.Outputs, grafanaURL)
	case "grafana:oss/dashboard:Dashboard":
		return p.buildDashboardURL(req.Outputs)
	case "grafana:alerting/ruleGroup:RuleGroup":
		return p.buildRuleGroupURL(req.Inputs, grafanaURL)
	case "grafana:alerting/contactPoint:ContactPoint":
		return p.buildContactPointURL(grafanaURL)
	case "grafana:alerting/muteTiming:MuteTiming":
		return p.buildMuteTimingURL(grafanaURL)
	case "grafana:alerting/notificationPolicy:NotificationPolicy":
		return p.buildNotificationPolicyURL(grafanaURL)
	case "grafana:cloud/accessPolicy:AccessPolicy":
		return p.buildCloudAccessPolicyURL(req.StackConfig, req.ProgramConfig)
	case "grafana:cloud/accessPolicyToken:AccessPolicyToken":
		return p.buildCloudAccessPolicyURL(req.StackConfig, req.ProgramConfig)
	case "grafana:cloud/stackServiceAccount:StackServiceAccount":
		return p.buildStackServiceAccountURL(req.Outputs, grafanaURL)
	case "grafana:cloud/stackServiceAccountToken:StackServiceAccountToken":
		return p.buildStackServiceAccountTokenURL(req.Inputs, grafanaURL)
	default:
		return "", nil
	}
}

func (p *GrafanaPlugin) buildEscalationChainURL(outputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	id := outputs["id"]
	if id == "" {
		return "", errEscalationChainIDMissing
	}
	return grafanaURL + irmAppPath + "/escalations/" + id, nil
}

func (p *GrafanaPlugin) buildEscalationURL(inputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	chainID := inputs["escalationChainId"]
	if chainID == "" {
		return "", errEscalationChainIDMissing
	}
	return grafanaURL + irmAppPath + "/escalations/" + chainID, nil
}

func (p *GrafanaPlugin) buildIntegrationURL(outputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	id := outputs["id"]
	if id == "" {
		return "", errIntegrationIDMissing
	}
	return grafanaURL + irmAppPath + "/integrations/" + id, nil
}

func (p *GrafanaPlugin) buildOnCallShiftURL(grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	return grafanaURL + irmAppPath + "/schedules", nil
}

func (p *GrafanaPlugin) buildRouteURL(inputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	integrationID := inputs["integrationId"]
	if integrationID == "" {
		return "", errIntegrationIDMissing
	}
	return grafanaURL + irmAppPath + "/integrations/" + integrationID, nil
}

func (p *GrafanaPlugin) buildScheduleURL(outputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	id := outputs["id"]
	if id == "" {
		return "", errScheduleIDMissing
	}
	return grafanaURL + irmAppPath + "/schedules/" + id, nil
}

func (p *GrafanaPlugin) buildTeamURL(outputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	teamUID := outputs["teamUid"]
	if teamUID == "" {
		return "", errTeamUIDMissing
	}
	return grafanaURL + "/org/teams/edit/" + teamUID, nil
}

func (p *GrafanaPlugin) buildRuleGroupURL(inputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	folderUID := inputs["folderUid"]
	if folderUID == "" {
		return "", errRuleGroupFolderMissing
	}
	name := inputs["name"]
	if name == "" {
		return "", errRuleGroupNameMissing
	}
	return grafanaURL + "/alerting/grafana/namespaces/" + folderUID + "/groups/" + name + "/view", nil
}

func (p *GrafanaPlugin) buildContactPointURL(grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	return grafanaURL + "/alerting/notifications", nil
}

func (p *GrafanaPlugin) buildMuteTimingURL(grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	return grafanaURL + "/alerting/routes?tab=time_intervals", nil
}

func (p *GrafanaPlugin) buildNotificationPolicyURL(grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	return grafanaURL + "/alerting/routes", nil
}

func (p *GrafanaPlugin) buildCloudAccessPolicyURL(stackConfig, programConfig map[string]string) (string, error) {
	orgSlug := stackConfig["grafana:cloudOrgSlug"]
	if orgSlug == "" {
		orgSlug = programConfig["grafana:cloudOrgSlug"]
	}
	if orgSlug != "" {
		return "https://grafana.com/orgs/" + orgSlug + "/access-policies", nil
	}
	return "https://grafana.com/orgs/access-policies", nil
}

func (p *GrafanaPlugin) buildStackServiceAccountURL(outputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	id := outputs["id"]
	if id != "" {
		if idx := strings.Index(id, ":"); idx != -1 {
			id = id[idx+1:]
		}
	}
	if id != "" {
		return grafanaURL + "/org/serviceaccounts/" + id, nil
	}
	return grafanaURL + "/org/serviceaccounts", nil
}

func (p *GrafanaPlugin) buildStackServiceAccountTokenURL(inputs map[string]string, grafanaURL string) (string, error) {
	if grafanaURL == "" {
		return "", errGrafanaURLNotConfigured
	}
	serviceAccountID := inputs["serviceAccountId"]
	if serviceAccountID != "" {
		if idx := strings.Index(serviceAccountID, ":"); idx != -1 {
			serviceAccountID = serviceAccountID[idx+1:]
		}
	}
	if serviceAccountID != "" {
		return grafanaURL + "/org/serviceaccounts/" + serviceAccountID, nil
	}
	return grafanaURL + "/org/serviceaccounts", nil
}

func (p *GrafanaPlugin) buildDashboardURL(outputs map[string]string) (string, error) {
	url := outputs["url"]
	if url == "" {
		return "", errDashboardURLMissing
	}
	return url, nil
}
