package util

import (
	"context"
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

)

// CreateRoute creates a Route for a given service in OpenShift
func CreateRoute(routeClient *routeclientset.Clientset, namespace, routeName, serviceName string, targetPort interface{}, hostname string) error {
	var targetPortValue intstr.IntOrString

	// Check if targetPort is a string or an int and handle accordingly
	switch v := targetPort.(type) {
	case int:
		targetPortValue = intstr.FromInt(v)
	case string:
		targetPortValue = intstr.FromString(v)
	default:
		return fmt.Errorf("unsupported type for targetPort: %T", targetPort)
	}

	// Define the route object
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			Host: hostname, // Optional hostname, empty means it will be auto-assigned
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: serviceName, // Route to the given service
			},
			Port: &routev1.RoutePort{
				TargetPort: targetPortValue, // Use the targetPort here
			},
		},
	}

	// Create the Route in OpenShift
	_, err := routeClient.RouteV1().Routes(namespace).Create(context.TODO(), route, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Route: %v", err)
	}

	return nil
}

// GetRouteURL returns the full URL for the given route
func GetRouteURL(routeClient *routeclientset.Clientset, namespace, routeName string) (string, error) {
	route, err := routeClient.RouteV1().Routes(namespace).Get(context.TODO(), routeName, metav1.GetOptions{})
	if err != nil {
		LogError("Failed to retrieve Route %s: %v", routeName, err)
		return "", err
	}

	if route.Spec.Host == "" {
		return "", fmt.Errorf("Route %s has no assigned host", routeName)
	}

	url := fmt.Sprintf("http://%s", route.Spec.Host)
	LogInfo("Route URL for %s: %s", routeName, url)
	return url, nil
}

// DeleteRoute deletes an OpenShift route
func DeleteRoute(routeClient *routeclientset.Clientset, namespace, routeName string) error {
	err := routeClient.RouteV1().Routes(namespace).Delete(context.TODO(), routeName, metav1.DeleteOptions{})
	if err != nil {
		LogError("Failed to delete Route %s: %v", routeName, err)
		return err
	}

	LogInfo("Successfully deleted Route %s", routeName)
	return nil
}
