package streams

// Struct representing the node of a radix trie
type RaxNode struct {
	IsEndOfEntry bool              // Is the current node the end of an entry or is it just a connecting node
	Children     map[byte]*RaxNode // To store the edges in the trie
	Prefix       []byte            // The common prefix of all children of this node
	Entry        *StreamEntry      // nil if this is not end of entry, else pointer to struct
}

// Struct representing a radix trie
type Rax struct {
	Root *RaxNode // Root node of trie
}

// Struct representing the edge of a radix trie (used to backtrack over trie)
type TrieEdge struct {
	parent *RaxNode
	node   *RaxNode
	edge   byte
}

func (r *Rax) Insert(s []byte, entry *StreamEntry) {
	node := r.Root
	i := 0

	// Start traversing over the string
	for i < len(s) {
		// Check if current node has a child with the i'th char
		child, ok := node.Children[s[i]]
		if !ok {
			// If child doesn't Exist, create a new child node
			newNode := &RaxNode{
				IsEndOfEntry: true,  // This node does represent the end of an entry
				Children:     nil,   // Child node has no children
				Prefix:       s[i:], // Prefix of the child node is the remaining string that we haven't traversed
				Entry:        entry,
			}
			// Make an edge b/w the current node and child node
			node.Children[s[i]] = newNode
			// We have inserted the element, break
			break
		}

		// Child Exists!!
		node = child // Descend the trie
		prefixLen := MaxCommonStringLen(s[i:], node.Prefix)
		i += prefixLen // Increment pointer over string

		// New entry has a common prefix with current node, Split:
		if prefixLen < len(node.Prefix) {
			// oldNode stores the entries AFTER the common prefix (from the existing node)
			oldNode := &RaxNode{
				IsEndOfEntry: node.IsEndOfEntry,
				Children:     node.Children,
				Prefix:       node.Prefix[prefixLen:], // Remaining part of OLD prefix
				Entry:        node.Entry,
			}

			// Update current node to be the split point
			node.Prefix = node.Prefix[:prefixLen]
			node.Children = make(map[byte]*RaxNode)
			node.Children[oldNode.Prefix[0]] = oldNode

			// Determine if current split point is end of entry
			if i == len(s) {
				// The input string ends exactly at the split point
				node.IsEndOfEntry = true
				node.Entry = entry
			} else {
				// Input string continues beyond split point - create new node for it
				node.IsEndOfEntry = false
				node.Entry = nil
				newNode := &RaxNode{
					IsEndOfEntry: true,
					Children:     nil,
					Prefix:       s[i:],
					Entry:        entry,
				}
				node.Children[s[i]] = newNode
			}
			break
		}
	}
}

func (r *Rax) SearchExact(s []byte) *RaxNode {
	node := r.Root
	i := 0

	// Start traversing over the string
	for i < len(s) {
		// Check if current node has a child with the i'th char
		child, ok := node.Children[s[i]]
		if !ok {
			// The entry doesn't exist, returning false
			return nil
		}

		node = child // Descend the trie
		prefixLen := MaxCommonStringLen(s[i:], node.Prefix)

		// If at any point the common prefix bw the input and the node differs:
		// the input doesn't exist in the trie, hence return false
		if prefixLen != len(node.Prefix) {
			return nil
		}

		// All checks are done, increment to the next node
		i += prefixLen

		// Node found after satisfying all conditions
		if i == len(s) {
			if node.IsEndOfEntry {
				return node
			} else {
				return nil
			}
		}
	}
	// Node not found
	return nil
}

func (r *Rax) Delete(s []byte) bool {
	// Maintain a stack of all the edges we are traversing on
	var stack []TrieEdge

	node := r.Root
	i := 0

	// Start traversing over the string
	for i < len(s) {
		// See if child exists for the current node with the edge of the i'th char
		child, ok := node.Children[s[i]]
		if !ok {
			// Child doesn't exist, we didn't delete anything...return
			return false
		}

		prefixLen := MaxCommonStringLen(s[i:], child.Prefix)
		if prefixLen != len(child.Prefix) {
			// the length of common prefix b/w the remaining string and the stored prefix
			// didn't match, hence the entry doesn't exist...returning false
			return false
		}

		// All good so far, add current edge to stack
		stack = append(stack, TrieEdge{
			parent: node,
			node:   child,
			edge:   s[i],
		})

		node = child // Descend the trie
		i += prefixLen
	}

	// After the loop, if the node isn't the end of an entry...
	// then we didn't delete anything, return false
	if !node.IsEndOfEntry {
		return false
	}

	// Unmark terminal
	node.IsEndOfEntry = false

	// Backtrack on the traversed nodes for cleanup
	for len(stack) > 0 {
		f := stack[len(stack)-1]     // Stack.top()
		stack = stack[:len(stack)-1] // Stack.pop()

		n := f.node

		// Case 1 - node still needed:
		// Current node is the end of an entry OR parent of another node
		if n.IsEndOfEntry || len(n.Children) > 1 {
			break
		}

		// Case 2 - merge with single child
		// We can compress the space by merging the single child with the current node
		if len(n.Children) == 1 {
			// For the single child of the node
			for _, child := range n.Children {
				// In the prefix of the current node, add the prefix of the child
				n.Prefix = append(n.Prefix, child.Prefix...)
				// Determine if the current node is the end of an entry based on child
				n.IsEndOfEntry = child.IsEndOfEntry
				// Transfer children
				n.Children = child.Children
			}
			break
		}

		// Case 3 - remove empty node
		delete(f.parent.Children, f.edge)
	}

	return true
}

func leftmost(n *RaxNode) *RaxNode {
	if n == nil {
		return nil
	}
	if n.IsEndOfEntry {
		return n
	}
	if len(n.Children) == 0 {
		return nil
	}

	// pick smallest edge
	var minKey int = 256
	var next *RaxNode
	for k, c := range n.Children {
		if int(k) < minKey {
			minKey = int(k)
			next = c
		}
	}
	return leftmost(next)
}

func (r *Rax) SeekGE(s []byte) *RaxNode {
	var stack []TrieEdge
	node := r.Root
	i := 0

	for i < len(s) {
		child, ok := node.Children[s[i]]
		if !ok {
			// No exact match for this character.
			// Find the first child of 'node' that is > s[i]
			var minKey int = 256
			var nextNode *RaxNode
			for k, c := range node.Children {
				if int(k) > int(s[i]) && int(k) < minKey {
					minKey = int(k)
					nextNode = c
				}
			}
			if nextNode != nil {
				return leftmost(nextNode)
			}
			// No larger child, backtrack
			break
		}

		prefixLen := MaxCommonStringLen(s[i:], child.Prefix)
		if prefixLen < len(child.Prefix) {
			// Prefix mismatch.
			// If the first differing byte in child.Prefix is > s[i+prefixLen], then this child is GE.
			if i+prefixLen < len(s) && child.Prefix[prefixLen] > s[i+prefixLen] {
				return leftmost(child)
			}
			// Otherwise, this child is smaller than s, so we need a larger sibling of this child.
			// We'll break and let the backtracking handle it.
			stack = append(stack, TrieEdge{parent: node, node: child, edge: s[i]})
			break
		}

		stack = append(stack, TrieEdge{parent: node, node: child, edge: s[i]})
		node = child
		i += prefixLen
	}

	if i == len(s) {
		if node.IsEndOfEntry {
			return node
		}
		// Not an entry, find leftmost in subtree
		if len(node.Children) > 0 {
			return leftmost(node)
		}
	}

	// Backtrack to find larger sibling
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		parent := top.parent
		currEdge := top.edge

		var minKey int = 256
		var nextNode *RaxNode

		for k, c := range parent.Children {
			if int(k) > int(currEdge) && int(k) < minKey {
				minKey = int(k)
				nextNode = c
			}
		}

		if nextNode != nil {
			return leftmost(nextNode)
		}
	}

	return nil
}

func (r *Rax) Successor(s []byte) *RaxNode {
	var stack []TrieEdge
	node := r.Root
	i := 0

	// Descend to the node
	for i < len(s) {
		child, ok := node.Children[s[i]]
		if !ok {
			return nil // Should not happen if s is an existing key
		}
		stack = append(stack, TrieEdge{parent: node, node: child, edge: s[i]})
		node = child
		i += len(child.Prefix)
	}

	// If node has children, the successor is the leftmost entry in the children's subtrees
	if len(node.Children) > 0 {
		var minKey int = 256
		var nextNode *RaxNode
		for k, c := range node.Children {
			if int(k) < minKey {
				minKey = int(k)
				nextNode = c
			}
		}
		return leftmost(nextNode)
	}

	// Otherwise, backtrack to find a larger sibling
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		parent := top.parent
		currEdge := top.edge

		var minKey int = 256
		var nextNode *RaxNode

		for k, c := range parent.Children {
			if int(k) > int(currEdge) && int(k) < minKey {
				minKey = int(k)
				nextNode = c
			}
		}

		if nextNode != nil {
			return leftmost(nextNode)
		}
	}

	return nil
}

func MaxCommonStringLen(a []byte, b []byte) int {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}
