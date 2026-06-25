package seed

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// challenge is an auto-gradable coding exercise: the learner writes code against
// `prompt`/`starter`, and `test` (hidden) is compiled alongside it by the checker.
type challenge struct {
	Slug, Title, Prompt, Starter, Test string
	Step, Position                     int
}

// challenges is the authored catalog. Both starter and test are `package solution`.
var challenges = []challenge{
	{
		Slug: "reverse-ints", Step: 1, Position: 1,
		Title:  "Reverse a slice",
		Prompt: "Implement `Reverse(s []int) []int` that returns a NEW slice with the elements of s in reverse order. The input slice must NOT be modified.",
		Starter: `package solution

// Reverse returns a new slice with the elements of s reversed.
// It must not modify s.
func Reverse(s []int) []int {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func TestReverse(t *testing.T) {
	in := []int{1, 2, 3}
	got := Reverse(in)
	if want := []int{3, 2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Reverse([1 2 3]) = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(in, []int{1, 2, 3}) {
		t.Fatalf("input slice was modified to %v", in)
	}
}

func TestReverseEmpty(t *testing.T) {
	if got := Reverse([]int{}); len(got) != 0 {
		t.Fatalf("Reverse([]) = %v, want empty", got)
	}
}
`,
	},
	{
		Slug: "word-count", Step: 1, Position: 2,
		Title:  "Word frequency counter",
		Prompt: "Implement `WordCount(s string) map[string]int` that returns how many times each whitespace-separated word appears in s. Example: WordCount(\"go go run\") == {\"go\":2, \"run\":1}.",
		Starter: `package solution

// WordCount returns the frequency of each whitespace-separated word in s.
func WordCount(s string) map[string]int {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func TestWordCount(t *testing.T) {
	got := WordCount("go go run")
	want := map[string]int{"go": 2, "run": 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WordCount = %v, want %v", got, want)
	}
}

func TestWordCountEmpty(t *testing.T) {
	if got := WordCount("   "); len(got) != 0 {
		t.Fatalf("WordCount(blank) = %v, want empty", got)
	}
}
`,
	},
	{
		Slug: "binary-search", Step: 2, Position: 1,
		Title:  "Binary search",
		Prompt: "Implement `Search(a []int, target int) int` that returns the index of target in the SORTED slice a, or -1 if it's absent. Must run in O(log n).",
		Starter: `package solution

// Search returns the index of target in the sorted slice a, or -1 if absent.
func Search(a []int, target int) int {
	// your code here
	return -1
}
`,
		Test: `package solution

import "testing"

func TestSearch(t *testing.T) {
	a := []int{1, 3, 5, 7, 9}
	cases := map[int]int{1: 0, 5: 2, 9: 4, 4: -1, 0: -1, 10: -1}
	for target, want := range cases {
		if got := Search(a, target); got != want {
			t.Fatalf("Search(%v, %d) = %d, want %d", a, target, got, want)
		}
	}
}

func TestSearchEmpty(t *testing.T) {
	if got := Search([]int{}, 1); got != -1 {
		t.Fatalf("Search([], 1) = %d, want -1", got)
	}
}
`,
	},
	{
		Slug: "generic-stack", Step: 2, Position: 2,
		Title:  "Generic LIFO stack",
		Prompt: "Implement a generic `Stack[T any]` with `Push(v T)`, `Pop() (T, bool)` (false when empty), and `Len() int`. Pushing 1,2,3 then popping three times must yield 3,2,1.",
		Starter: `package solution

// Stack is a generic LIFO stack.
type Stack[T any] struct {
	// your fields
}

func (s *Stack[T]) Push(v T) {
	// your code here
}

// Pop removes and returns the top element; ok is false if the stack is empty.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	return zero, false
}

func (s *Stack[T]) Len() int {
	return 0
}
`,
		Test: `package solution

import "testing"

func TestStack(t *testing.T) {
	var s Stack[int]
	if s.Len() != 0 {
		t.Fatalf("empty Len() = %d, want 0", s.Len())
	}
	s.Push(1)
	s.Push(2)
	s.Push(3)
	if s.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", s.Len())
	}
	for _, want := range []int{3, 2, 1} {
		got, ok := s.Pop()
		if !ok || got != want {
			t.Fatalf("Pop() = %d,%v want %d,true", got, ok, want)
		}
	}
	if _, ok := s.Pop(); ok {
		t.Fatalf("Pop() on empty returned ok=true")
	}
}
`,
	},

	// ---- interview fundamentals (map to common LeetCode/NeetCode problems) ----
	{
		Slug: "fizz-buzz", Step: 1, Position: 3,
		Title:  "FizzBuzz",
		Prompt: "Implement `FizzBuzz(n int) []string` returning the FizzBuzz sequence for 1..n: \"Fizz\" for multiples of 3, \"Buzz\" for 5, \"FizzBuzz\" for both, otherwise the number as a string.",
		Starter: `package solution

// FizzBuzz returns the FizzBuzz sequence for 1..n.
func FizzBuzz(n int) []string {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func TestFizzBuzz(t *testing.T) {
	got := FizzBuzz(15)
	want := []string{"1", "2", "Fizz", "4", "Buzz", "Fizz", "7", "8", "Fizz", "Buzz", "11", "Fizz", "13", "14", "FizzBuzz"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FizzBuzz(15) = %v, want %v", got, want)
	}
}
`,
	},
	{
		Slug: "two-sum", Step: 2, Position: 3,
		Title:  "Two Sum",
		Prompt: "Implement `TwoSum(nums []int, target int) []int` returning the indices (ascending) of the two numbers that add up to target. Exactly one solution exists. Aim for O(n).",
		Starter: `package solution

// TwoSum returns the indices of the two numbers adding to target.
func TwoSum(nums []int, target int) []int {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func TestTwoSum(t *testing.T) {
	cases := []struct {
		nums   []int
		target int
		want   []int
	}{
		{[]int{2, 7, 11, 15}, 9, []int{0, 1}},
		{[]int{3, 2, 4}, 6, []int{1, 2}},
		{[]int{3, 3}, 6, []int{0, 1}},
	}
	for _, c := range cases {
		if got := TwoSum(c.nums, c.target); !reflect.DeepEqual(got, c.want) {
			t.Fatalf("TwoSum(%v, %d) = %v, want %v", c.nums, c.target, got, c.want)
		}
	}
}
`,
	},
	{
		Slug: "contains-duplicate", Step: 2, Position: 4,
		Title:  "Contains Duplicate",
		Prompt: "Implement `ContainsDuplicate(nums []int) bool` returning true if any value appears at least twice.",
		Starter: `package solution

// ContainsDuplicate reports whether any value appears at least twice.
func ContainsDuplicate(nums []int) bool {
	// your code here
	return false
}
`,
		Test: `package solution

import "testing"

func TestContainsDuplicate(t *testing.T) {
	cases := []struct {
		nums []int
		want bool
	}{
		{[]int{1, 2, 3, 1}, true},
		{[]int{1, 2, 3, 4}, false},
		{[]int{}, false},
		{[]int{1, 1, 1, 3, 3, 4, 3, 2, 4, 2}, true},
	}
	for _, c := range cases {
		if got := ContainsDuplicate(c.nums); got != c.want {
			t.Fatalf("ContainsDuplicate(%v) = %v, want %v", c.nums, got, c.want)
		}
	}
}
`,
	},
	{
		Slug: "valid-anagram", Step: 2, Position: 5,
		Title:  "Valid Anagram",
		Prompt: "Implement `IsAnagram(s, t string) bool` returning true if t is an anagram of s.",
		Starter: `package solution

// IsAnagram reports whether t is an anagram of s.
func IsAnagram(s, t string) bool {
	// your code here
	return false
}
`,
		Test: `package solution

import "testing"

func TestIsAnagram(t *testing.T) {
	cases := []struct {
		s, t string
		want bool
	}{
		{"anagram", "nagaram", true},
		{"rat", "car", false},
		{"", "", true},
		{"a", "ab", false},
	}
	for _, c := range cases {
		if got := IsAnagram(c.s, c.t); got != c.want {
			t.Fatalf("IsAnagram(%q, %q) = %v, want %v", c.s, c.t, got, c.want)
		}
	}
}
`,
	},
	{
		Slug: "valid-parentheses", Step: 2, Position: 6,
		Title:  "Valid Parentheses",
		Prompt: "Implement `IsValid(s string) bool` for a string of '()[]{}' returning true iff brackets are correctly closed and nested. (Classic stack problem.)",
		Starter: `package solution

// IsValid reports whether the bracket string is correctly closed and nested.
func IsValid(s string) bool {
	// your code here
	return false
}
`,
		Test: `package solution

import "testing"

func TestIsValid(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"()", true},
		{"()[]{}", true},
		{"(]", false},
		{"([)]", false},
		{"{[]}", true},
		{"", true},
		{"(", false},
		{"]", false},
	}
	for _, c := range cases {
		if got := IsValid(c.s); got != c.want {
			t.Fatalf("IsValid(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}
`,
	},
	{
		Slug: "reverse-linked-list", Step: 2, Position: 7,
		Title:  "Reverse Linked List",
		Prompt: "Implement `ReverseList(head *ListNode) *ListNode` reversing a singly linked list and returning the new head.",
		Starter: `package solution

// ListNode is a singly-linked list node.
type ListNode struct {
	Val  int
	Next *ListNode
}

// ReverseList reverses the list and returns the new head.
func ReverseList(head *ListNode) *ListNode {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func buildList(vals ...int) *ListNode {
	dummy := &ListNode{}
	cur := dummy
	for _, v := range vals {
		cur.Next = &ListNode{Val: v}
		cur = cur.Next
	}
	return dummy.Next
}

func listToSlice(h *ListNode) []int {
	out := []int{}
	for h != nil {
		out = append(out, h.Val)
		h = h.Next
	}
	return out
}

func TestReverseList(t *testing.T) {
	got := listToSlice(ReverseList(buildList(1, 2, 3, 4, 5)))
	if want := []int{5, 4, 3, 2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ReverseList = %v, want %v", got, want)
	}
	if ReverseList(nil) != nil {
		t.Fatalf("ReverseList(nil) should be nil")
	}
}
`,
	},
	{
		Slug: "merge-two-sorted-lists", Step: 2, Position: 8,
		Title:  "Merge Two Sorted Lists",
		Prompt: "Implement `MergeTwoLists(l1, l2 *ListNode) *ListNode` merging two sorted lists into one sorted list.",
		Starter: `package solution

// ListNode is a singly-linked list node.
type ListNode struct {
	Val  int
	Next *ListNode
}

// MergeTwoLists merges two sorted lists into one sorted list.
func MergeTwoLists(l1, l2 *ListNode) *ListNode {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func buildList(vals ...int) *ListNode {
	dummy := &ListNode{}
	cur := dummy
	for _, v := range vals {
		cur.Next = &ListNode{Val: v}
		cur = cur.Next
	}
	return dummy.Next
}

func listToSlice(h *ListNode) []int {
	out := []int{}
	for h != nil {
		out = append(out, h.Val)
		h = h.Next
	}
	return out
}

func TestMergeTwoLists(t *testing.T) {
	got := listToSlice(MergeTwoLists(buildList(1, 2, 4), buildList(1, 3, 4)))
	if want := []int{1, 1, 2, 3, 4, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeTwoLists = %v, want %v", got, want)
	}
	if MergeTwoLists(nil, nil) != nil {
		t.Fatalf("merge(nil,nil) should be nil")
	}
	if got := listToSlice(MergeTwoLists(nil, buildList(0))); !reflect.DeepEqual(got, []int{0}) {
		t.Fatalf("merge(nil,[0]) = %v, want [0]", got)
	}
}
`,
	},
	{
		Slug: "max-subarray", Step: 2, Position: 9,
		Title:  "Maximum Subarray (Kadane)",
		Prompt: "Implement `MaxSubArray(nums []int) int` returning the largest sum of any contiguous non-empty subarray. Aim for O(n) (Kadane's algorithm).",
		Starter: `package solution

// MaxSubArray returns the maximum contiguous subarray sum.
func MaxSubArray(nums []int) int {
	// your code here
	return 0
}
`,
		Test: `package solution

import "testing"

func TestMaxSubArray(t *testing.T) {
	cases := []struct {
		nums []int
		want int
	}{
		{[]int{-2, 1, -3, 4, -1, 2, 1, -5, 4}, 6},
		{[]int{1}, 1},
		{[]int{5, 4, -1, 7, 8}, 23},
		{[]int{-3, -2, -5}, -2},
	}
	for _, c := range cases {
		if got := MaxSubArray(c.nums); got != c.want {
			t.Fatalf("MaxSubArray(%v) = %d, want %d", c.nums, got, c.want)
		}
	}
}
`,
	},
	{
		Slug: "climbing-stairs", Step: 2, Position: 10,
		Title:  "Climbing Stairs (DP)",
		Prompt: "Implement `ClimbStairs(n int) int` — the number of distinct ways to climb n steps taking 1 or 2 steps at a time. (Fibonacci-shaped DP.)",
		Starter: `package solution

// ClimbStairs returns the number of distinct ways to climb n steps.
func ClimbStairs(n int) int {
	// your code here
	return 0
}
`,
		Test: `package solution

import "testing"

func TestClimbStairs(t *testing.T) {
	cases := map[int]int{1: 1, 2: 2, 3: 3, 5: 8, 10: 89}
	for n, want := range cases {
		if got := ClimbStairs(n); got != want {
			t.Fatalf("ClimbStairs(%d) = %d, want %d", n, got, want)
		}
	}
}
`,
	},
	{
		Slug: "binary-tree-inorder", Step: 2, Position: 11,
		Title:  "Binary Tree Inorder Traversal",
		Prompt: "Implement `InorderTraversal(root *TreeNode) []int` returning the in-order traversal of a binary tree.",
		Starter: `package solution

// TreeNode is a binary tree node.
type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

// InorderTraversal returns the in-order traversal values.
func InorderTraversal(root *TreeNode) []int {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"testing"
)

func TestInorderTraversal(t *testing.T) {
	// tree: 1 with right child 2 whose left child is 3  => inorder [1,3,2]
	root := &TreeNode{Val: 1, Right: &TreeNode{Val: 2, Left: &TreeNode{Val: 3}}}
	if got := InorderTraversal(root); !reflect.DeepEqual(got, []int{1, 3, 2}) {
		t.Fatalf("InorderTraversal = %v, want [1 3 2]", got)
	}
	if got := InorderTraversal(nil); len(got) != 0 {
		t.Fatalf("InorderTraversal(nil) = %v, want empty", got)
	}
}
`,
	},
	{
		Slug: "valid-bst", Step: 2, Position: 12,
		Title:  "Validate Binary Search Tree",
		Prompt: "Implement `IsValidBST(root *TreeNode) bool` returning true iff the tree is a valid BST (every node's entire left subtree < node < entire right subtree).",
		Starter: `package solution

// TreeNode is a binary tree node.
type TreeNode struct {
	Val   int
	Left  *TreeNode
	Right *TreeNode
}

// IsValidBST reports whether the tree is a valid binary search tree.
func IsValidBST(root *TreeNode) bool {
	// your code here
	return false
}
`,
		Test: `package solution

import "testing"

func TestIsValidBST(t *testing.T) {
	valid := &TreeNode{Val: 2, Left: &TreeNode{Val: 1}, Right: &TreeNode{Val: 3}}
	if !IsValidBST(valid) {
		t.Fatalf("expected valid BST")
	}
	// 3 lives in the right subtree of 5 but is < 5 => invalid
	invalid := &TreeNode{Val: 5,
		Left:  &TreeNode{Val: 1},
		Right: &TreeNode{Val: 4, Left: &TreeNode{Val: 3}, Right: &TreeNode{Val: 6}}}
	if IsValidBST(invalid) {
		t.Fatalf("expected invalid BST")
	}
	if !IsValidBST(nil) {
		t.Fatalf("empty tree is a valid BST")
	}
}
`,
	},
	{
		Slug: "top-k-frequent", Step: 2, Position: 13,
		Title:  "Top K Frequent Elements",
		Prompt: "Implement `TopKFrequent(nums []int, k int) []int` returning the k most frequent elements (any order). Aim for better than O(n log n) — a heap or bucket sort.",
		Starter: `package solution

// TopKFrequent returns the k most frequent elements, in any order.
func TopKFrequent(nums []int, k int) []int {
	// your code here
	return nil
}
`,
		Test: `package solution

import (
	"reflect"
	"sort"
	"testing"
)

func TestTopKFrequent(t *testing.T) {
	got := TopKFrequent([]int{1, 1, 1, 2, 2, 3}, 2)
	sort.Ints(got)
	if want := []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("TopKFrequent = %v, want %v (any order)", got, want)
	}
	got2 := TopKFrequent([]int{1}, 1)
	if !reflect.DeepEqual(got2, []int{1}) {
		t.Fatalf("TopKFrequent([1],1) = %v, want [1]", got2)
	}
}
`,
	},
	{
		Slug: "number-of-islands", Step: 2, Position: 14,
		Title:  "Number of Islands (graph/BFS)",
		Prompt: "Implement `NumIslands(grid [][]byte) int` — count islands in a grid where '1' is land and '0' is water, connected 4-directionally.",
		Starter: `package solution

// NumIslands counts 4-directionally connected groups of '1' (land) cells.
func NumIslands(grid [][]byte) int {
	// your code here
	return 0
}
`,
		Test: `package solution

import "testing"

func TestNumIslands(t *testing.T) {
	cases := []struct {
		grid [][]byte
		want int
	}{
		{[][]byte{{'1', '1', '0'}, {'1', '0', '0'}, {'0', '0', '1'}}, 2},
		{[][]byte{{'0', '0'}, {'0', '0'}}, 0},
		{[][]byte{{'1', '1', '1'}, {'0', '1', '0'}, {'1', '1', '1'}}, 1},
		{[][]byte{}, 0},
	}
	for _, c := range cases {
		if got := NumIslands(c.grid); got != c.want {
			t.Fatalf("NumIslands = %d, want %d", got, c.want)
		}
	}
}
`,
	},
}

// SyncChallenges upserts the challenge catalog into code_challenges (by slug).
func SyncChallenges(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	for _, c := range challenges {
		_, err := pool.Exec(ctx, `
			INSERT INTO code_challenges (step_id, slug, title, prompt, starter_code, test_code, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (slug) DO UPDATE SET
				step_id=EXCLUDED.step_id, title=EXCLUDED.title, prompt=EXCLUDED.prompt,
				starter_code=EXCLUDED.starter_code, test_code=EXCLUDED.test_code, position=EXCLUDED.position`,
			c.Step, c.Slug, c.Title, c.Prompt, c.Starter, c.Test, c.Position)
		if err != nil {
			return 0, err
		}
	}
	return len(challenges), nil
}
