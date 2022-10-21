package core

import (
	"reflect"

	"k8s.io/alibaba-load-balancer-controller/pkg/model/alb/core/graph"

	"github.com/pkg/errors"
)

// Manager presents a resource graph, where resources can depend on each other.
type Manager interface {
	// stackID returns a unique ID for stack.
	StackID() StackID

	// Add a resource into stack.
	AddResource(res Resource) error

	// Add a dependency relationship between resources.
	AddDependency(dependee Resource, depender Resource) error

	// ListResources list all resources for specific type.
	// pResourceSlice must be a pointer to a slice of resources, which will be filled.
	ListResources(pResourceSlice interface{}) error

	// TopologicalTraversal visits resources in stack in topological order.
	TopologicalTraversal(visitor ResourceVisitor) error
}

// NewDefaultManager constructs new stack.
func NewDefaultManager(stackID StackID) *defaultManager {
	return &defaultManager{
		stackID: stackID,

		resources:     make(map[graph.ResourceUID]Resource),
		resourceGraph: graph.NewDefaultResourceGraph(),
	}
}

var _ Manager = &defaultManager{}

// default implementation for stack.
type defaultManager struct {
	stackID StackID

	resources     map[graph.ResourceUID]Resource
	resourceGraph graph.ResourceGraph
}

func (s *defaultManager) StackID() StackID {
	return s.stackID
}

// Add a resource.
func (s *defaultManager) AddResource(res Resource) error {
	resUID := s.computeResourceUID(res)
	if _, ok := s.resources[resUID]; ok {
		return errors.Errorf("resource already exists, type: %v, id: %v", res.Type(), res.ID())
	}
	s.resources[resUID] = res
	s.resourceGraph.AddNode(resUID)
	return nil
}

// Add a dependency relationship between resources.
func (s *defaultManager) AddDependency(dependee Resource, depender Resource) error {
	dependeeResUID := s.computeResourceUID(dependee)
	dependerResUID := s.computeResourceUID(depender)
	if _, ok := s.resources[dependeeResUID]; !ok {
		return errors.Errorf("dependee resource didn't exists, type: %v, id: %v", dependee.Type(), dependee.ID())
	}
	if _, ok := s.resources[dependerResUID]; !ok {
		return errors.Errorf("depender resource didn't exists, type: %v, id: %v", depender.Type(), depender.ID())
	}
	s.resourceGraph.AddEdge(dependeeResUID, dependerResUID)
	return nil
}

// ListResources list all resources for specific type.
// pResourceSlice must be a pointer to a slice of resources, which will be filled.
func (s *defaultManager) ListResources(pResourceSlice interface{}) error {
	v := reflect.ValueOf(pResourceSlice)
	if v.Kind() != reflect.Ptr {
		return errors.New("pResourceSlice must be pointer to resource slice")
	}
	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return errors.New("pResourceSlice must be pointer to resource slice")
	}
	resType := v.Type().Elem()
	var resForType []Resource
	for resID, res := range s.resources {
		if resID.ResType == resType {
			resForType = append(resForType, res)
		}
	}
	v.Set(reflect.MakeSlice(v.Type(), len(resForType), len(resForType)))
	for i := range resForType {
		v.Index(i).Set(reflect.ValueOf(resForType[i]))
	}
	return nil
}

func (s *defaultManager) TopologicalTraversal(visitor ResourceVisitor) error {
	return graph.TopologicalTraversal(s.resourceGraph, func(uid graph.ResourceUID) error {
		return visitor.Visit(s.resources[uid])
	})
}

// computeResourceUID returns the UID for resources.
func (s *defaultManager) computeResourceUID(res Resource) graph.ResourceUID {
	return graph.ResourceUID{
		ResType: reflect.TypeOf(res),
		ResID:   res.ID(),
	}
}
