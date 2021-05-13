package data

import (
	"fmt"

	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	roleContextCluster       = "cluster"
	roleContextNamespace     = "namespace"
	roleContextLabelKey      = "harvesterhci.io/roleContext"
	builtInLabelKey          = "harvesterhci.io/builtIn"
	displayNameAnnotationKey = "harvesterhci.io/displayName"
	namePrefix               = "harvester-"
	nameLabelKey             = "harvesterhci.io/name"
	aggregateToLabelPrefix   = "rbac.authorization.k8s.io/aggregate-to-"
)

type clusterRoleBuilder struct {
	previous                *clusterRoleBuilder
	next                    *clusterRoleBuilder
	name                    string
	displayName             string
	context                 string
	builtin                 bool
	aggregationClusterRoles []string
	aggregateToClusterRoles []string
	rules                   []*ruleBuilder
}

func (crb *clusterRoleBuilder) String() string {
	return fmt.Sprintf("%s (%s): %s", crb.displayName, crb.name, crb.rules)
}

func newClusterRoleBuilder() *clusterRoleBuilder {
	return &clusterRoleBuilder{
		builtin: true,
	}
}

func (crb *clusterRoleBuilder) addRole(displayName, name string, context string) *clusterRoleBuilder {
	if crb.name == "" {
		crb.name = name
		crb.displayName = displayName
		crb.context = context
		return crb
	}

	if crb.next != nil {
		return crb.next.addRole(displayName, name, context)
	}
	crb.next = newClusterRoleBuilder()
	crb.next.name = name
	crb.next.displayName = displayName
	crb.next.context = context
	crb.next.previous = crb

	return crb.next
}

func (crb *clusterRoleBuilder) addRule() *ruleBuilder {
	r := &ruleBuilder{
		rb: crb,
	}
	crb.rules = append(crb.rules, r)
	return r
}

func (crb *clusterRoleBuilder) setAggregationClusterRoles(names ...string) *clusterRoleBuilder {
	crb.aggregationClusterRoles = names
	return crb
}

func (crb *clusterRoleBuilder) setAggregateToClusterRoles(names ...string) *clusterRoleBuilder {
	crb.aggregateToClusterRoles = names
	return crb
}

func (crb *clusterRoleBuilder) first() *clusterRoleBuilder {
	if crb.previous == nil {
		return crb
	}
	return crb.previous.first()
}

func (crb *clusterRoleBuilder) policyRules() []v1.PolicyRule {
	prs := make([]v1.PolicyRule, len(crb.rules))
	for i, r := range crb.rules {
		prs[i] = r.toPolicyRule()
	}
	return prs
}

func (crb *clusterRoleBuilder) build() []runtime.Object {
	var result []runtime.Object
	for cr := crb.first(); cr != nil; cr = cr.next {
		clusterRole := &v1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: namePrefix + cr.name,
				Labels: map[string]string{
					builtInLabelKey:     "true",
					roleContextLabelKey: cr.context,
				},
				Annotations: map[string]string{
					displayNameAnnotationKey: cr.displayName,
				},
			},
			Rules: cr.policyRules(),
		}
		if len(cr.aggregateToClusterRoles) > 0 {
			for _, name := range cr.aggregateToClusterRoles {
				clusterRole.Labels[aggregateToLabelPrefix+name] = "true"
			}
		}
		if len(cr.aggregationClusterRoles) > 0 {
			clusterRole.AggregationRule = &v1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      nameLabelKey,
								Operator: metav1.LabelSelectorOpIn,
								Values:   cr.aggregationClusterRoles,
							},
						},
					},
				},
			}
		}
		result = append(result, clusterRole)
	}
	return result
}

type ruleBuilder struct {
	rb             *clusterRoleBuilder
	_verbs         []string
	_resources     []string
	_resourceNames []string
	_apiGroups     []string
	_urls          []string
}

func (r *ruleBuilder) String() string {
	return fmt.Sprintf("apigroups: %v, resource: %v, resourceNames: %v, nonResourceURLs %v, verbs: %v", r._apiGroups, r._resources, r._resourceNames, r._urls, r._verbs)
}

func (r *ruleBuilder) verbs(v ...string) *ruleBuilder {
	r._verbs = append(r._verbs, v...)
	return r
}

func (r *ruleBuilder) resources(rc ...string) *ruleBuilder {
	r._resources = append(r._resources, rc...)
	return r
}

func (r *ruleBuilder) resourceNames(rn ...string) *ruleBuilder {
	r._resourceNames = append(r._resourceNames, rn...)
	return r
}

func (r *ruleBuilder) apiGroups(a ...string) *ruleBuilder {
	r._apiGroups = append(r._apiGroups, a...)
	return r
}

func (r *ruleBuilder) nonResourceURLs(u ...string) *ruleBuilder {
	r._urls = append(r._urls, u...)
	return r
}

func (r *ruleBuilder) addRole(displayName, name string, context string) *clusterRoleBuilder {
	return r.rb.addRole(displayName, name, context)
}

func (r *ruleBuilder) addRule() *ruleBuilder {
	return r.rb.addRule()
}

func (r *ruleBuilder) setAggregationClusterRoles(names ...string) *clusterRoleBuilder {
	return r.rb.setAggregationClusterRoles(names...)
}

func (r *ruleBuilder) setAggregateToClusterRoles(names ...string) *clusterRoleBuilder {
	return r.rb.setAggregateToClusterRoles(names...)
}

func (r *ruleBuilder) toPolicyRule() v1.PolicyRule {
	return v1.PolicyRule{
		APIGroups:       r._apiGroups,
		Resources:       r._resources,
		ResourceNames:   r._resourceNames,
		NonResourceURLs: r._urls,
		Verbs:           r._verbs,
	}
}
