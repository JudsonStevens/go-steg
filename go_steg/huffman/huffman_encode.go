package huffman

import "sort"

/*
Currently a work in progress.
Eventually we'll want to use this to encode the data we are going to embed in a Huffman tree, which will allow us to
store much more data than before.
*/

// Node defines the node we'll use to build our Huffman tree - the value of the leaf nodes
// are the integer values of the channels, and the values of the tree/root nodes are the frequencies
// When navigating down the tree, you find the shortest path to the value you need - left nodes result
// in a 0, right nodes result in a 1
// More information - https://www.techiedelight.com/huffman-coding/
type Node struct {
	Parent *Node
	Left   *Node
	Right  *Node
	Count  int
	Value  int32
}

// SortNodes is a type defined to give us a slice to store the pointers for each node
type SortNodes []*Node

// Len simply returns the length of the Node slice - the type in the parameter
// before the method name allows us to call that method on the "class" SortNodes
// like SortNodes.Len()
// We define these in order to let the sort package use them with the context
// of our Nodes instead of something else to sort - sort.Stable takes in an interface/type
// and then uses our rewritten methods to do the actual action
func (sn SortNodes) Len() int { return len(sn) }

// Less will determine if the node at the first index is less than the node at the next index
func (sn SortNodes) Less(i, j int) bool { return sn[i].Count < sn[j].Count }

// Swap will swap two nodes
func (sn SortNodes) Swap(i, j int) { sn[i], sn[j] = sn[j], sn[i] }

// BuildTree builds a Huffman tree out of individual nodes
func BuildTree(leaves []*Node) *Node {
	//Do one sort to get the slice sorted least to most
	sort.Stable(SortNodes(leaves))

	return BuildPreSortedTree(leaves)
}

// BuildPreSortedTree creates a Huffman tree from a pre-sorted slice of leaf nodes.
// The nodes must be sorted by their Count value in ascending order.
//
// The algorithm works by repeatedly:
// 1. Taking the two nodes with lowest counts
// 2. Creating a parent node with those as children
// 3. Inserting the parent back into the sorted list
//
// Example: For leaf nodes with counts [1, 2, 4, 7, 8], the tree would look like:
//
//	       22
//	      /  \
//	     14   8
//	    / \
//	   3   11
//	  / \  / \
//	 1   2 4  7
//
// To find a value, traverse the tree using 0 for left, 1 for right.
// Example: The code "011" leads to value 7 (left, right, right)
//
// Real-world example using the word "hello":
// Character frequencies: (h:1, e:1, l:2, o:1)
// Sorted by frequency: [h:1, e:1, o:1, l:2]
// Results in tree:
//	        5
//	       / \
//	      2   3
//	     /     \
//	    h     2(l,l)
//	   / \
//	  e   o
//
// Resulting codes:
// h = 00
// e = 010
// o = 011
// l = 1
func BuildPreSortedTree(leaves []*Node) *Node {
	if len(leaves) == 0 {
		return nil
	}

	for len(leaves) > 1 {
		// Take the two lowest-count nodes as children
		leftChild := leaves[0]
		rightChild := leaves[1]

		// Create their parent node, combining their counts
		parentNode := &Node{
			Left:  leftChild,
			Right: rightChild,
			Count: leftChild.Count + rightChild.Count,
		}

		// Link children back to their parent
		leftChild.Parent = parentNode
		rightChild.Parent = parentNode

		// Find where to insert the new parent node to maintain sorted order:
		// 1. Look at remaining nodes (index 2 onwards)
		ls := leaves[2:]
		// 2. Find first position where node count >= parent count
		idx := sort.Search(len(ls), func(i int) bool { return ls[i].Count >= parentNode.Count })
		idx += 2 // Adjust index to account for the two nodes we removed

		// Shift existing nodes and insert parent to maintain sorted order
		copy(leaves[1:], leaves[2:idx])
		leaves[idx-1] = parentNode
		// Remove the first node (we used two nodes to make one)
		leaves = leaves[1:]
	}

	// After combining all nodes, only the root remains
	return leaves[0]
}

// ReturnCode will return the binary code for that node by walking up the tree
// using its parent, etc. Left children are a 0, right are a 1
func (n *Node) ReturnCode() (r uint64, bits byte) {
	for parent := n.Parent; parent != nil; n, parent = parent, parent.Parent {
		//If we are on a right child, push a bit onto r
		//The |= operator is a compound operator, equal to r = r | 1
		//This will always activate the last bit (switch it to 1)
		// then we left shift
		//Not sure why we use bits as a byte here, an int could function the same way
		if parent.Right == n {
			r |= 1 << bits
		}
		bits++
	}
	//This blank return will return both return types specified above
	return
}
