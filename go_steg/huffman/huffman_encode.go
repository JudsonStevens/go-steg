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

// Len simply returns the length of the Node slice - the type in the paran
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

// BuildPreSortedTree will return a Huffman tree after the leaves have been sorted
// by the method above. Since the slice of Nodes is already sorted by Count,
// we know that we are starting with the smallest elements by count. That lets
// us start building our tree from smallest to largest
// For example - [1, 2, 4, 7, 8] -> grab the first two frequencies (the values of the node)
//
//	       22
//	      /  \
//	     14   8
//	    / \
//	  3     11
//	 / \   /  \
//	1   2 4    7
//
// In this instance, if I wanted to get the value of 011, that would be 7 - start at 22, 0 means go left
// arriving at 14, 1 means go right arriving at 11, and 1 means go right, arriving at 7
func BuildPreSortedTree(leaves []*Node) *Node {
	if len(leaves) == 0 {
		return nil
	}

	for len(leaves) > 1 {
		// Start with the first and second elements
		left, right := leaves[0], leaves[1]
		parentCount := left.Count + right.Count
		parent := &Node{Left: left, Right: right, Count: parentCount}
		left.Parent = parent
		right.Parent = parent

		//This portion of the function will insert the new node into the slice
		// and keep it sorted correctly
		//Set ls equal to the leaves slice starting at the second index
		ls := leaves[2:]
		//Set the index variable equal to the return of the Search function
		//This function will return the index of the value that first matches the statement
		// inside the function. Basically we are finding the first node in the slice
		// that has a count greater than or equal to our current parentCount. In that case
		// we will insert into that index
		idx := sort.Search(len(ls), func(i int) bool { return ls[i].Count >= parentCount })
		//We increase idx by 2 in order to be able to insert correctly later
		idx += 2

		//Copy will copy the source (leaves[2:idx]) to the destination (leaves[1:]) and copy
		// over those indexes while maintaining all the indexes after
		//For example - copy([1, 2, 3, 4], [3, 4]) => [1, 3, 4, 4]
		copy(leaves[1:], leaves[2:idx])
		leaves[idx-1] = parent
		leaves = leaves[1:]
	}
	//Once we're done building the tree, the last node will be the root node
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
