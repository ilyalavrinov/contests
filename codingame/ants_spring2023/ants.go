package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

func cmdBeacon(cellId int, strength int) string {
	return fmt.Sprintf("BEACON %d %d", cellId, strength)
}

func cmdLine(cellIdFrom, cellIdTo int, strength int) string {
	return fmt.Sprintf("LINE %d %d %d", cellIdFrom, cellIdTo, strength)
}

func cmdWait() string {
	return "WAIT"
}

func cmdMessage(msg string) string {
	return fmt.Sprintf("MESSAGE %s", msg)
}

const (
	CELLTYPE_NOTHING int = 0
	CELLTYPE_EGG     int = 1
	CELLTYPE_CRYSTAL int = 2

	CELLTYPE_MYBASE int = 10
)

type Cell struct {
	index         int
	cellType      int
	resourceCount int

	myAnts  int
	oppAnts int

	neighbours []*Cell
}

type Field struct {
	numberOfCells int
	cells         map[int]*Cell

	myBases    []*Cell
	enemyBases []*Cell

	myScore  int
	oppScore int

	interestingCells map[int]*Cell
	distances        map[int]map[int]int

	cellsWithCrystals []*Cell
	cellsWithEggs     []*Cell

	myAntsCount int
}

func ScanNewField(scanner *bufio.Scanner) Field {
	var field Field

	var inputs []string

	scanner.Scan()
	fmt.Sscan(scanner.Text(), &field.numberOfCells)

	field.cells = make(map[int]*Cell, field.numberOfCells)
	field.interestingCells = make(map[int]*Cell)
	neighbourLists := make(map[int][]int)

	for i := 0; i < field.numberOfCells; i++ {
		// _type: 0 for empty, 1 for eggs, 2 for crystal
		// initialResources: the initial amount of eggs/crystals on this cell
		// neigh0: the index of the neighbouring cell for each direction
		var _type, initialResources, neigh0, neigh1, neigh2, neigh3, neigh4, neigh5 int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &_type, &initialResources, &neigh0, &neigh1, &neigh2, &neigh3, &neigh4, &neigh5)

		cell := &Cell{
			index:         i,
			cellType:      _type,
			resourceCount: initialResources,
		}
		field.cells[i] = cell

		if _type == CELLTYPE_CRYSTAL {
			field.cellsWithCrystals = append(field.cellsWithCrystals, cell)
			field.interestingCells[i] = cell
		} else if _type == CELLTYPE_EGG {
			field.cellsWithEggs = append(field.cellsWithEggs, cell)
			field.interestingCells[i] = cell
		}

		neighbourLists[i] = append(neighbourLists[i], neigh0, neigh1, neigh2, neigh3, neigh4, neigh5)
	}

	for cellIx, neighbours := range neighbourLists {
		cell := field.cells[cellIx]
		for _, neighIx := range neighbours {
			if neighIx == -1 {
				continue
			}
			cell.neighbours = append(cell.neighbours, field.cells[neighIx])
		}
	}

	var numberOfBases int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &numberOfBases)

	scanner.Scan()
	inputs = strings.Split(scanner.Text(), " ")
	for i := 0; i < numberOfBases; i++ {
		myBaseIndex, _ := strconv.ParseInt(inputs[i], 10, 32)
		field.myBases = append(field.myBases, field.cells[int(myBaseIndex)])
		field.cells[int(myBaseIndex)].cellType = CELLTYPE_MYBASE
		field.interestingCells[int(myBaseIndex)] = field.cells[int(myBaseIndex)]
	}
	scanner.Scan()
	inputs = strings.Split(scanner.Text(), " ")
	for i := 0; i < numberOfBases; i++ {
		oppBaseIndex, _ := strconv.ParseInt(inputs[i], 10, 32)
		field.enemyBases = append(field.enemyBases, field.cells[int(oppBaseIndex)])
	}

	calcInterestingDistances(&field)

	return field
}

func calcInterestingDistances(f *Field) {
	f.distances = make(map[int]map[int]int)
	for i := range f.interestingCells {
		f.distances[i] = bfs(i, f)
	}
}

func bfs(from int, f *Field) map[int]int {
	cell := f.cells[from]
	nextLevel := cell.neighbours
	distances := make(map[int]int)
	distances[from] = 0
	dist := 1
	for _, n := range nextLevel {
		distances[n.index] = dist
	}

	calcDistBfs := func(frontier []*Cell) {
		for _, cell := range frontier {
			for _, n := range cell.neighbours {
				if _, found := distances[n.index]; found {
					continue
				}
				distances[n.index] = dist
				nextLevel = append(nextLevel, n)
			}
		}
	}

	for len(nextLevel) > 0 {
		dist++
		currentLevel := make([]*Cell, 0, len(nextLevel))
		for _, n := range nextLevel {
			currentLevel = append(currentLevel, n)
		}
		nextLevel = make([]*Cell, 0)
		calcDistBfs(currentLevel)
	}

	return distances
}

func (f *Field) ScanNewTurn(scanner *bufio.Scanner) {
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &f.myScore, &f.oppScore)

	for i := 0; i < f.numberOfCells; i++ {
		cell := f.cells[i]

		scanner.Scan()
		fmt.Sscan(scanner.Text(), &cell.resourceCount, &cell.myAnts, &cell.oppAnts)
		f.myAntsCount += cell.myAnts
	}
}

func (f *Field) cleanUpEmptyResources() bool {
	result := false
	for i, cell := range f.interestingCells {
		if (cell.cellType == CELLTYPE_CRYSTAL || cell.cellType == CELLTYPE_EGG) && cell.resourceCount == 0 {
			delete(f.interestingCells, i)
			result = true
		}
	}
	return result
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)
	field := ScanNewField(scanner)

	mst := calculateMST(&field)

	firstGet := true

	for {
		field.ScanNewTurn(scanner)
		if field.cleanUpEmptyResources() {
			firstGet = false
			mst = calculateMST(&field)
		}

		var cmds []string
		if !firstGet {
			for from, tos := range mst {
				for _, to := range tos {
					cmds = append(cmds, cmdLine(from, to, 1))
				}
			}
		} else {
			for _, base := range field.myBases {
				nodes := mst[base.index]
				for _, index := range nodes {
					if field.cells[index].cellType == CELLTYPE_EGG {
						cmds = append(cmds, cmdLine(base.index, index, 1))
					}
				}
			}
		}

		/*		if len(field.myBases) != 1 {
					for from, tos := range mst {
						for _, to := range tos {
							cmds = append(cmds, cmdLine(from, to, 1))
						}
					}
				} else {
					//cmds = greedyLimitedGather(&field, mst)
					cmds = longestChain(&field, mst)
					cmds = append(cmds, cmdMessage("ONEBASE"))
				}
		*/
		printCmds(cmds...)
	}
}

func printCmds(cmds ...string) {
	result := make([]string, 0, len(cmds))
	if len(cmds) == 0 {
		result = append(result, cmdWait())
	} else {
		for _, cmd := range cmds {
			result = append(result, cmd)
		}
	}
	fmt.Println(strings.Join(result, ";"))
}

func calculateMST(f *Field) map[int][]int {
	unvisited := make(map[int]struct{}, len(f.interestingCells))
	for _, cell := range f.interestingCells {
		unvisited[cell.index] = struct{}{}
	}
	visited := make(map[int]struct{}, len(f.interestingCells)+len(f.myBases))
	/*
		for _, base := range f.myBases {
			nodes = append(nodes, base)
		}
	*/
	edges := make(map[int][]int)

	newIteration := func() (int, int) {
		minDist := math.MaxInt
		bestFrom := -1
		bestTo := -1
		for from := range visited {
			for to := range unvisited {
				d := f.distances[from][to]
				if d < minDist {
					minDist = d
					bestFrom = from
					bestTo = to
				}
			}
		}
		return bestFrom, bestTo
	}

	for _, base := range f.myBases {
		visited[base.index] = struct{}{}
	}
	for len(unvisited) != 0 {
		from, to := newIteration()
		edges[from] = append(edges[from], to)
		visited[to] = struct{}{}
		delete(unvisited, to)
	}

	edgesSorted := make(map[int][]int, len(edges))
	for from, tos := range edges {
		sort.Slice(tos, func(i, j int) bool {
			return tos[i] < tos[j]
		})
		edgesSorted[from] = tos
	}

	return edgesSorted
}

func greedyLimitedGather(f *Field, mst map[int][]int) []string {
	fromsToCheck := make(map[int]bool)
	fromsToCheck[f.myBases[0].index] = true
	remainingAnts := f.myAntsCount

	iterate := func() (int, int, int) {
		minDistFrom := -1
		minDistTo := -1
		minDist := math.MaxInt
		for from := range fromsToCheck {
			for _, to := range mst[from] {
				if fromsToCheck[to] {
					// already seen
					continue
				}
				dist := f.distances[from][to]
				if dist+1 > remainingAnts {
					continue
				}

				if dist < minDist {
					minDist = dist
					minDistFrom = from
					minDistTo = to
				}
			}
		}
		return minDistFrom, minDistTo, minDist
	}

	cmds := make([]string, 0)
	for {
		if len(cmds) >= 5 {
			break
		}
		from, to, dist := iterate()
		remainingAnts -= dist + 1
		fromsToCheck[to] = true
		if to == -1 {
			break
		}
		cmds = append(cmds, cmdLine(from, to, 1))
	}
	return cmds
}

func longestChain(f *Field, mst map[int][]int) []string {
	base := f.myBases[0].index
	allPaths := dfs(base, mst, []int{base})

	var longestPath []int
	longestPathDist := 0
	for _, path := range allPaths {
		if len(path) > longestPathDist {
			longestPathDist = len(path)
			longestPath = path
		}
	}

	cmds := make([]string, 0)
	from := longestPath[0]
	for i := 1; i < len(longestPath); i++ {
		to := longestPath[i]
		cmds = append(cmds, cmdLine(from, to, 1))
		from = to
	}
	return cmds
}

func dfs(cur int, mst map[int][]int, curPath []int) [][]int {
	nodes := mst[cur]
	paths := make([][]int, 0)
	if nodes != nil {
		for _, node := range nodes {
			newPath := append(curPath, node)
			paths = append(paths, dfs(node, mst, newPath)...)
		}
	} else {
		paths = [][]int{curPath}
	}
	return paths
}
