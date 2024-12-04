package k8s

import (
	"context"
	"fmt"
	"os"

	"github.com/supporttools/RancherResourceScanner/pkg/logging"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var log = logging.SetupLogging()

type ResourceCheckResult struct {
	Namespace      string
	Resource       string
	Name           string
	Issue          string
	AdditionalInfo string
}

// ConnectToCluster connects to the Kubernetes cluster and returns both *kubernetes.Clientset and dynamic.Interface
func ConnectToCluster(kubeconfig string) (*kubernetes.Clientset, dynamic.Interface, error) {
	var config *rest.Config
	var err error

	// Use in-cluster config if available
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("error creating in-cluster config: %v", err)
		}
	} else {
		// Fall back to kubeconfig
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
			if kubeconfig == "" {
				kubeconfig = fmt.Sprintf("%s/.kube/config", os.Getenv("HOME"))
			}
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, nil, fmt.Errorf("error creating kubeconfig: %v", err)
		}
	}

	// Create the *kubernetes.Clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating clientset: %v", err)
	}

	// Create the dynamic.Interface
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating dynamic client: %v", err)
	}

	return clientset, dynamicClient, nil
}

// VerifyAccessToCluster verifies the connection to the Kubernetes cluster by listing nodes.
func VerifyAccessToCluster(clientset *kubernetes.Clientset) error {
	log.Infoln("Verifying access to the Kubernetes cluster...")
	ctx := context.TODO()
	listOptions := v1.ListOptions{}

	_, err := clientset.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("error listing nodes: %v", err)
	}

	log.Infoln("Access to the Kubernetes cluster verified successfully.")
	return nil
}

func GetNamespaces(clientset *kubernetes.Clientset) ([]string, error) {
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	namespaces := make([]string, len(namespaceList.Items))
	for i, namespace := range namespaceList.Items {
		namespaces[i] = namespace.GetName()
	}
	return namespaces, nil
}

// GetNamespaceScopedResources returns a list of namespaced resources as []schema.GroupVersionResource
func GetNamespaceScopedResources(clientset *kubernetes.Clientset) ([]schema.GroupVersionResource, error) {
	discoveryClient := clientset.Discovery()
	apiResourceLists, err := discoveryClient.ServerPreferredNamespacedResources()
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			// Handle partial discovery errors
			fmt.Printf("Partial discovery error: %v\n", err)
		} else {
			return nil, fmt.Errorf("error fetching namespaced resources: %v", err)
		}
	}

	var resources []schema.GroupVersionResource
	for _, apiResourceList := range apiResourceLists {
		groupVersion, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("error parsing GroupVersion %s: %v", apiResourceList.GroupVersion, err)
		}
		for _, apiResource := range apiResourceList.APIResources {
			if apiResource.Namespaced {
				resources = append(resources, groupVersion.WithResource(apiResource.Name))
			}
		}
	}

	return resources, nil
}

// GetClusterScopedResources fetches all cluster-scoped resources
func GetClusterScopedResources(clientset *kubernetes.Clientset) ([]schema.GroupVersionResource, error) {
	discoveryClient := clientset.Discovery()
	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			fmt.Printf("Partial discovery error: %v\n", err)
		} else {
			return nil, fmt.Errorf("error fetching cluster-scoped resources: %v", err)
		}
	}

	var resources []schema.GroupVersionResource
	for _, apiResourceList := range apiResourceLists {
		groupVersion, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("error parsing GroupVersion %s: %v", apiResourceList.GroupVersion, err)
		}

		for _, apiResource := range apiResourceList.APIResources {
			if !apiResource.Namespaced { // Only include cluster-scoped resources
				resources = append(resources, groupVersion.WithResource(apiResource.Name))
			}
		}
	}

	return resources, nil
}

func GetNamespacedObjects(clientset *kubernetes.Clientset) ([]schema.GroupVersionResource, error) {
	log.Infoln("Fetching namespaced API resources...")
	apiResourceList, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		log.Errorf("Error fetching namespaced API resources: %v", err)
		return nil, err
	}
	log.Debugln("API Resources: ", apiResourceList)

	var gv schema.GroupVersion
	objects := make([]schema.GroupVersionResource, 0)
	for _, apiResources := range apiResourceList {
		gv, _ = schema.ParseGroupVersion(apiResources.GroupVersion) // Use = for assignment
		for _, apiResource := range apiResources.APIResources {
			if apiResource.Namespaced {
				// Create a GroupVersionResource with the API version included
				object := gv.WithResource(apiResource.Name)
				objects = append(objects, object)
			}
		}
	}
	log.Debugln("Namespaced Objects: ", objects)
	return objects, nil
}

// GetNamespaceObjects retrieves the list of object names for a specific resource in a namespace.
func GetNamespaceObjects(dynamicClient dynamic.Interface, ns string, resource schema.GroupVersionResource, apiVersion string) ([]string, error) {
	// List the objects for the given resource
	resourceList, err := dynamicClient.Resource(resource).Namespace(ns).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing objects for resource %s in namespace %s: %v", resource.Resource, ns, err)
	}

	// Extract the names of the objects
	objectNames := make([]string, len(resourceList.Items))
	for i, obj := range resourceList.Items {
		objectNames[i] = obj.GetName()
	}

	return objectNames, nil
}

func GetAPIVersionForResource(clientset *kubernetes.Clientset, resource schema.GroupVersionResource) (string, error) {
	// Get the API resource
	apiResource, err := clientset.Discovery().ServerResourcesForGroupVersion(resource.GroupVersion().String())
	if err != nil {
		return "", err
	}
	// Get the API version
	apiVersion := apiResource.APIResources[0].Version
	return apiVersion, nil
}

func ScanNamespaceResources(clientset *kubernetes.Clientset, dynamicClient dynamic.Interface) ([]ResourceCheckResult, error) {
	var results []ResourceCheckResult

	// Get namespace-scoped resources
	log.Debug("Fetching namespace-scoped resources...")
	resources, err := GetNamespaceScopedResources(clientset)
	if err != nil {
		log.Errorf("Error fetching namespace-scoped resources: %v", err)
		return nil, err
	}
	log.Debugf("Found %d namespace-scoped resources", len(resources))

	// Get namespaces
	log.Debug("Fetching namespaces...")
	namespaces, err := GetNamespaces(clientset)
	if err != nil {
		log.Errorf("Error fetching namespaces: %v", err)
		return nil, err
	}
	log.Debugf("Found %d namespaces", len(namespaces))

	// Iterate through namespaces and resources
	for _, ns := range namespaces {
		log.Debugf("Processing namespace: %s", ns)
		for _, resource := range resources {
			// Check if the resource supports the "list" verb
			if !resourceSupportsList(clientset, resource) {
				log.Debugf("Skipping resource %s as it does not support 'list'", resource.Resource)
				continue
			}

			log.Debugf("Processing resource: %s in namespace: %s", resource.Resource, ns)

			// Fetch resource objects in the namespace
			objects, err := GetNamespaceObjects(dynamicClient, ns, resource, resource.GroupVersion().String())
			if err != nil {
				log.Errorf("Error fetching objects for resource %s in namespace %s: %v", resource.Resource, ns, err)
				continue
			}
			log.Debugf("Found %d objects for resource %s in namespace %s", len(objects), resource.Resource, ns)

			// Check each object for issues
			for _, objectName := range objects {
				log.Debugf("Checking object: %s/%s of resource %s", ns, objectName, resource.Resource)

				obj, err := dynamicClient.Resource(resource).Namespace(ns).Get(context.TODO(), objectName, v1.GetOptions{})
				if err != nil {
					log.Errorf("Error fetching object %s/%s of resource %s: %v", ns, objectName, resource.Resource, err)
					continue
				}

				// Check for stuck finalizers
				finalizerIssues := CheckStuckFinalizers(obj)
				if len(finalizerIssues) > 0 {
					log.Errorf("Critical issue: Detected %d stuck finalizers for object: %s/%s", len(finalizerIssues), ns, objectName)
				}
				results = append(results, finalizerIssues...)

				// Check for invalid ownerReferences
				ownerRefIssues := CheckInvalidOwnerReferences(dynamicClient, obj, ns)
				if len(ownerRefIssues) > 0 {
					log.Errorf("Critical issue: Detected %d invalid ownerReferences for object: %s/%s", len(ownerRefIssues), ns, objectName)
					results = append(results, ownerRefIssues...)
				}
			}
		}

	}

	log.Infof("Scanning completed: Found %d issues", len(results))
	return results, nil
}

// resourceSupportsList checks if the resource supports the "list" operation.
func resourceSupportsList(clientset *kubernetes.Clientset, resource schema.GroupVersionResource) bool {
	discoveryClient := clientset.Discovery()
	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		log.Errorf("Error fetching API resources to check for 'list' support: %v", err)
		return false
	}

	for _, apiResourceList := range apiResourceLists {
		if apiResourceList.GroupVersion == resource.GroupVersion().String() {
			for _, apiResource := range apiResourceList.APIResources {
				if apiResource.Name == resource.Resource {
					for _, verb := range apiResource.Verbs {
						if verb == "list" {
							return true
						}
					}
					return false
				}
			}
		}
	}
	return false
}

// CheckStuckFinalizers checks if an object has stuck finalizers.
func CheckStuckFinalizers(obj *unstructured.Unstructured) []ResourceCheckResult {
	var results []ResourceCheckResult

	// Check for stuck finalizers
	if len(obj.GetFinalizers()) > 0 && obj.GetDeletionTimestamp() != nil {
		log.Debugf("Detected stuck finalizer on object: Namespace=%s, Name=%s, Finalizers=%v, DeletionTimestamp=%v",
			obj.GetNamespace(), obj.GetName(), obj.GetFinalizers(), obj.GetDeletionTimestamp())

		results = append(results, ResourceCheckResult{
			Namespace: obj.GetNamespace(),
			Resource:  obj.GetKind(),
			Name:      obj.GetName(),
			Issue:     "Stuck finalizer",
			AdditionalInfo: fmt.Sprintf(
				"Finalizers: %v, DeletionTimestamp: %v",
				obj.GetFinalizers(),
				obj.GetDeletionTimestamp(),
			),
		})
	} else {
		log.Debugf("No stuck finalizers for object: Namespace=%s, Name=%s", obj.GetNamespace(), obj.GetName())
	}

	return results
}

// CheckInvalidOwnerReferences checks for invalid ownerReferences in an object.
func CheckInvalidOwnerReferences(dynamicClient dynamic.Interface, obj *unstructured.Unstructured, namespace string) []ResourceCheckResult {
	var results []ResourceCheckResult

	for _, owner := range obj.GetOwnerReferences() {
		if !OwnerExists(dynamicClient, owner, namespace) {
			log.Debugf("Detected invalid ownerReference on object %s/%s", obj.GetNamespace(), obj.GetName())
			results = append(results, ResourceCheckResult{
				Namespace: obj.GetNamespace(),
				Resource:  obj.GetKind(),
				Name:      obj.GetName(),
				Issue:     "Invalid ownerReference",
				AdditionalInfo: fmt.Sprintf(
					"OwnerReference: APIVersion=%s, Kind=%s, Name=%s, UID=%s",
					owner.APIVersion,
					owner.Kind,
					owner.Name,
					owner.UID,
				),
			})
		}
	}

	return results
}

// OwnerExists checks if the ownerReference exists in the cluster.
func OwnerExists(dynamicClient dynamic.Interface, owner v1.OwnerReference, namespace string) bool {
	gv, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		log.Errorf("Error parsing OwnerReference APIVersion: %v", err)
		return false
	}

	resourceClient := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: owner.Kind,
	})

	_, err = resourceClient.Namespace(namespace).Get(context.TODO(), owner.Name, v1.GetOptions{})
	return err == nil
}
