package globals

import (
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/event"
)

type PrefixFilterPredicate struct {
	Prefix string
}

func (p PrefixFilterPredicate) Create(e event.CreateEvent) bool {
	return strings.HasPrefix(e.Object.GetName(), p.Prefix)
}

func (p PrefixFilterPredicate) Delete(e event.DeleteEvent) bool {
	return strings.HasPrefix(e.Object.GetName(), p.Prefix)
}

func (p PrefixFilterPredicate) Update(e event.UpdateEvent) bool {
	return strings.HasPrefix(e.ObjectNew.GetName(), p.Prefix)
}

func (p PrefixFilterPredicate) Generic(e event.GenericEvent) bool {
	return strings.HasPrefix(e.Object.GetName(), p.Prefix)
}
