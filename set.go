// Goset is a thread safe SET data structure implementation
package goset

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type Set struct {
	m    map[interface{}]struct{}
	l    sync.RWMutex // we name it because we don't want to expose it
	kind reflect.Kind // runtime generics enforcement
}

// New creates and initialize a new Set. It's accept a variable number of
// arguments to populate the initial set. If nothing passed a Set with zero
// size is created.
func New(kind reflect.Kind, items ...interface{}) *Set {
	s := &Set{
		kind: kind,
		m: make(map[interface{}]struct{}), // struct{} doesn't take up space
	}

	s.Add(items...)
	return s
}

// Add includes the specified items (one or more) to the set. If passed nothing
// it silently returns.
func (s *Set) Add(items ...interface{}) error {
	if len(items) == 0 {
		return nil
	}
	if err := s.typecheck(items); err != nil {
		return err
	}

	s.l.Lock()
	defer s.l.Unlock()

	for _, item := range items {
		s.m[item] = struct{}{}
	}
	return nil
}

// Remove deletes the specified items from the set. If passed nothing it
// silently returns.
func (s *Set) Remove(items ...interface{}) error {
	if len(items) == 0 {
		return nil
	}
	if err := s.typecheck(items); err != nil {
		return err
	}

	s.l.Lock()
	defer s.l.Unlock()

	for _, item := range items {
		delete(s.m, item)
	}
	return nil
}

// Has looks for the existence of items passed. It returns false if nothing is
// passed. For multiple items it returns true only if all of  the items exist.
func (s *Set) Has(items ...interface{}) (bool, error) {
	// assume checked for empty item, which not exist
	if len(items) == 0 {
		return false, nil
	}
	if err := s.typecheck(items); err != nil {
		return false, err
	}

	s.l.RLock()
	defer s.l.RUnlock()

	for _, item := range items {
		if _, ok := s.m[item]; !ok {
			return false, nil
		}
	}
	return true, nil
}

// Size returns the number of items in a set.
func (s *Set) Size() int {
	s.l.RLock()
	defer s.l.RUnlock()
	return len(s.m)
}

// Clear removes all items from the set.
func (s *Set) Clear() {
	s.l.Lock()
	defer s.l.Unlock()
	s.m = make(map[interface{}]struct{})
}

// IsEmpty checks for emptiness of the set.
func (s *Set) IsEmpty() bool {
	return s.Size() == 0
}

// IsEqual test whether s and t are the same in size and have the same items.
func (s *Set) IsEqual(t *Set) (bool, error) {
	if err := s.typematch(t); err != nil {
		return false, err
	}

	if s.Size() != t.Size() {
		return false, nil
	}
	if u, _ := s.Union(t); s.Size() != u.Size() {
		return false, nil
	}
	return true, nil
}

// IsSubset tests t is a subset of s.
func (s *Set) IsSubset(t *Set) (bool, error) {
	if err := s.typematch(t); err != nil {
		return false, err
	}

	for _, item := range t.List() {
		if ok, _ := s.Has(item); !ok {
			return false, nil
		}
	}
	return true, nil
}

// IsSuperset tests if t is a superset of s.
func (s *Set) IsSuperset(t *Set) (bool, error) {
	return t.IsSubset(s)
}

// String representation of s
func (s *Set) String() string {
	t := make([]string, 0)
	for _, item := range s.List() {
		t = append(t, fmt.Sprintf("%v", item))
	}
	return fmt.Sprintf("[%s]", strings.Join(t, ", "))
}

// List returns a slice of all items
func (s *Set) List() []interface{} {
	s.l.RLock()
	defer s.l.RUnlock()
	list := make([]interface{}, 0)
	for item := range s.m {
		list = append(list, item)
	}
	return list
}

// Copy returns a new Set with a copy of s.
func (s *Set) Copy() *Set {
	return New(s.kind, s.List()...)
}

// Union is the merger of two sets. It returns a new set with the element in s
// and t combined.
func (s *Set) Union(t *Set) (*Set, error) {
	if err := s.typematch(t); err != nil {
		return nil, err
	}

	u := New(s.kind, t.List()...)
	for _, item := range s.List() {
		u.Add(item)
	}
	return u, nil
}

// Merge is like Union, however it modifies the current set it's applied on
// with the given t set.
func (s *Set) Merge(t *Set) error {
	if err := s.typematch(t); err != nil {
		return err
	}

	for _, item := range t.List() {
		s.Add(item)
	}
	return nil
}

// Separate removes the set items containing in t from set s. Please aware that
// it's not the opposite of Merge.
func (s *Set) Separate(t *Set) error {
	if err := s.typematch(t); err != nil {
		return err
	}

	for _, item := range t.List() {
		s.Remove(item)
	}
	return nil
}

// Intersection returns a new set which contains items which is in both s and t.
func (s *Set) Intersection(t *Set) (*Set, error) {
	if err := s.typematch(t); err != nil {
		return nil, err
	}

	u := New(s.kind)
	for _, item := range s.List() {
		if ok, _ := t.Has(item); ok {
			u.Add(item)
		}
	}
	for _, item := range t.List() {
		if ok, _ := s.Has(item); ok {
			u.Add(item)
		}
	}
	return u, nil
}

// Intersection returns a new set which contains items which are both s but not in t.
func (s *Set) Difference(t *Set) (*Set, error) {
	if err := s.typematch(t); err != nil {
		return nil, err
	}

	u := New(s.kind)
	for _, item := range s.List() {
		if ok, _ := t.Has(item); !ok {
			u.Add(item)
		}
	}
	return u, nil
}

// Symmetric returns a new set which s is the difference of items  which are in
// one of either, but not in both.
func (s *Set) SymmetricDifference(t *Set) (*Set, error) {
	if err := s.typematch(t); err != nil {
		return nil, err
	}

	u, _ := s.Difference(t)
	v, _ := t.Difference(s)
	res, _ := u.Union(v)
	return res, nil
}

// StringSlice is a helper function that returns a slice of strings of s. If
// the set contains mixed types of items only items of type string are returned.
func (s *Set) StringSlice() []string {
	slice := make([]string, 0)
	for _, item := range s.List() {
		v, ok := item.(string)
		if !ok {
			continue
		}

		slice = append(slice, v)
	}
	return slice
}

// IntSlice is a helper function that returns a slice of ints of s. If
// the set contains mixed types of items only items of type int are returned.
func (s *Set) IntSlice() []int {
	slice := make([]int, 0)
	for _, item := range s.List() {
		v, ok := item.(int)
		if !ok {
			continue
		}

		slice = append(slice, v)
	}
	return slice
}

func (s *Set) typematch(t *Set) error {
	if s.kind != t.kind {
		return fmt.Errorf("cannot perform the requested operation on mismatched sets; '%s' != '%s'", s.kind.String(), t.kind.String())
	}
	return nil
}

func (s *Set) typecheck(items ...interface{}) error {
	for _, item := range items {
		k := reflect.TypeOf(item).Kind()
		if k != s.kind {
			return fmt.Errorf("tried to insert value of kind '%s' into a set of kind '%s'", k.String(), s.kind.String())
		}
	}
	return nil
}
