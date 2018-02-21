package shape

import (
	"../shared"
	"fmt"
	"math"
	"strconv"
)

/*
type shared.Point2d struct {
	x int
	y int
}

type shared.Path struct {
	d          string
	fill       bool
	stroke     string
	vertexList []shared.Point2d
}
*/

// An arbitrary point outside the canvas
const INF = 10000000

// Checks if point q lies on Line p-r
func OnLine(p shared.Point2d, q shared.Point2d, r shared.Point2d) bool {
	if q.X <= int(math.Max(float64(p.X), float64(r.X))) && q.X >= int(math.Min(float64(p.X), float64(r.X))) &&
		q.Y <= int(math.Max(float64(p.Y), float64(r.Y))) && q.Y >= int(math.Min(float64(p.Y), float64(r.Y))) {
		return true
	}
	return false
}

// 0 = p, q, r are colinear
// 1 = p, q, r are clockwise
// 2 = p, q, r are counterclockwise
func Orientation(p shared.Point2d, q shared.Point2d, r shared.Point2d) int {
	val := (q.Y-p.Y)*(r.X-q.X) - (q.X-p.X)*(r.Y-q.Y)
	if val == 0 {
		return 0
	} else if val > 0 {
		return 1
	} else {
		return 2
	}
}

// Checks if 2 lines intersects
func IsLineIntersect(p1 shared.Point2d, q1 shared.Point2d, p2 shared.Point2d, q2 shared.Point2d) bool {
	o1 := Orientation(p1, q1, p2)
	o2 := Orientation(p1, q1, q2)
	o3 := Orientation(p2, q2, p1)
	o4 := Orientation(p2, q2, q1)

	// checks if o1 o2 have different Orientation, or o3 o4 have different Orientation
	if o1 != o2 && o3 != o4 {
		return true
	}

	// p1, q1 and p2 are colinear and p2 lies on line p1-q1
	if o1 == 0 && OnLine(p1, p2, q1) {
		return true
	}

	// p1, q1 and p2 are colinear and q2 lies on line p1-q1
	if o2 == 0 && OnLine(p1, q2, q1) {
		return true
	}

	// p2, q2 and p1 are colinear and p1 lies on line p2-q2
	if o3 == 0 && OnLine(p2, p1, q2) {
		return true
	}

	// p2, q2 and q1 are colinear and q1 lies on line p2-q2
	if o4 == 0 && OnLine(p2, q1, q2) {
		return true
	}

	return false
}

// Checks if a point is inside a given polygon (simple closed curve)
func IsPointInside(polygon []shared.Point2d, p shared.Point2d) bool {
	// If less than 3, its just a line not a polygon
	if len(polygon) < 3 {
		return false
	}

	// Create a point far away from p (out of the canvas)
	inf := shared.Point2d{X: INF, Y: p.Y}

	// Count intersections of line p-inf with the edges of the polygon
	var count int
	for i := 0; i < len(polygon)-1; i++ {
		// Check if the line from p-inf intersects with polygon[i]-polygon[i+1]
		if IsLineIntersect(polygon[i], polygon[i+1], p, inf) {
			// If p is colinear with i-i+1, and p lies on i-i+1, return true
			if Orientation(polygon[i], p, polygon[i+1]) == 0 {
				return OnLine(polygon[i], p, polygon[i+1])
			}
			// To avoid double counting on colinear Orientation
			// Handles cases where p intersects with the meeting point of the two lines in polygon
			if Orientation(polygon[i+1], p, inf) != 0 {
				count++
			}
		}
	}
	return (count%2 == 1)
}

// Check if the two paths intersect/overlap
func IsPathIntersect(vl1 []shared.Point2d, vl2 []shared.Point2d) bool {
	// If both vl1 vl2 are both just a line
	if len(vl1) == 2 && len(vl2) == 2 {
		return IsLineIntersect(vl1[0], vl1[1], vl2[0], vl2[1])
	}
	for i := 0; i < len(vl1); i++ {
		if IsPointInside(vl2, vl1[i]) {
			fmt.Println("Point ", vl1[i], "is inside vl2")
			return true
		}
	}
	for j := 0; j < len(vl2); j++ {
		if IsPointInside(vl1, vl2[j]) {
			fmt.Println("Point ", vl2[j], "is inside vl1")
			return true
		}
	}
	return false
}

// Calculates the area of the shape, only call this if Path is filled
// Area is rounded up
func GetArea(vertexList []shared.Point2d) (area int) {
	var sum int
	if (len(vertexList) < 3) {
		return 0
	}
	for i := 0; i < len(vertexList)-1; i++ {
		p1 := vertexList[i]
		p2 := vertexList[i+1]
		sum += (p1.X*p2.Y - p1.Y*p2.X)
	}
	area = int(math.Ceil(math.Abs(float64(sum / 2))))
	return area
}

// transform the svg string to a list of vertices
// assuming svg string draws a simple closed curve for filled shapes
// returns a list of points and length for transparent shape
// length is rounded up to the nearest int
func ShapeToVertexList(s string) (vertexList []shared.Point2d, length int) {
	var stringArray = StringToStringArray(s)
	var currentPosX = 0
	var currentPosY = 0
	var startPosX = 0
	var startPosY = 0
	var index = 0
	var l float64

	for index < len(stringArray) {
		switch stringArray[index] {
		case "M", "L", "m", "l":
			tempX, err := strconv.Atoi(stringArray[index+1])
			tempY, err := strconv.Atoi(stringArray[index+2])
			if err != nil {
				fmt.Println("Invalid SVG String")
				// TODO: return InvalidShapeSvgStringError
			}

			// Uppercase = absolute pos, lowercase = relative pos
			if stringArray[index] == "M" {
				startPosX = tempX
				startPosY = tempY
				currentPosX = tempX
				currentPosY = tempY
			} else if stringArray[index] == "m" {
				startPosX += tempX
				startPosY += tempY
				currentPosX += tempX
				currentPosY += tempY
			} else if stringArray[index] == "L" {
				// need to calculate the length of x and y
				x := math.Abs(float64(currentPosX - tempX))
				y := math.Abs(float64(currentPosY - tempY))
				l += math.Sqrt(x*x + y*y)
				currentPosX = tempX
				currentPosY = tempY
			} else {
				l += math.Sqrt(float64(tempX*tempX + tempY*tempY))
				currentPosX += tempX
				currentPosY += tempY
			}

			index += 3
		case "H", "h":
			tempX, err := strconv.Atoi(stringArray[index+1])
			if err != nil {
				fmt.Println("Invalid SVG String")
				// TODO: return InvalidShapeSvgStringError
			}

			if stringArray[index] == "H" {
				x := math.Abs(float64(currentPosX - tempX))
				l += x
				currentPosX = tempX
			} else {
				l += math.Abs(float64(tempX))
				currentPosX += tempX
			}

			index += 2
		case "V", "v":
			tempY, err := strconv.Atoi(stringArray[index+1])
			if err != nil {
				fmt.Println("Invalid SVG String")
				// TODO: return InvalidShapeSvgStringError
			}

			if stringArray[index] == "V" {
				y := math.Abs(float64(currentPosY - tempY))
				l += y
				currentPosY = tempY
			} else {
				l += math.Abs(float64(tempY))
				currentPosY += tempY
			}

			index += 2
		case "Z", "z":
			x := math.Abs(float64(currentPosX - startPosX))
			y := math.Abs(float64(currentPosY - startPosY))
			l += math.Sqrt(x*x + y*y)
			currentPosX = startPosX
			currentPosY = startPosY

			index++
		default:
			fmt.Println("unsupported svg command")
			// TODO: return InvalidShapeSvgStringError
			index++
		}
		// Adding a new vertex
		point := shared.Point2d{X: currentPosX, Y: currentPosY}
		vertexList = append(vertexList, point)
	}
	length = int(math.Ceil(l))
	return vertexList, length
}

// Removing spaces from the string and turning it into a string array
func StringToStringArray(s string) (stringArray []string) {
	var temp = ""
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			stringArray = append(stringArray, temp)
			temp = ""
		} else {
			temp += string(s[i])
		}
	}
	// Appending the last temp string
	stringArray = append(stringArray, temp)
	return stringArray
}

/*
func main() {
	// An example where one contains the other
	vertexList, length := shapeToVertexList("M 20 20 h 20 v 20 h -20 z")
	area := getArea(vertexList)
	fmt.Println("vertexList: ", vertexList, " lenght: ", length, " area: ", area)

	vertexList2, length2 := shapeToVertexList("M 0 0 h 100 v 100 l -100 -50 z")
	area2 := getArea(vertexList)
	fmt.Println("vertexList2: ", vertexList2, " lenght: ", length2, " area: ", area2)

	// Checks if vl & vl2 intersects
	intersect := isshared.PathIntersect(vertexList, vertexList2)
	fmt.Println("Is vl1 vl2 Intersect: ", intersect)

	// A convex example, they should not intersect
	vertexList3, length3 := shapeToVertexList("M 50 50 h 20 v 20 h -20 z")
	area3 := getArea(vertexList3)
	fmt.Println("vertexList3: ", vertexList3, " lenght: ", length3, " area: ", area3)

	vertexList4, length4 := shapeToVertexList("M 0 0 h 200 v 200 l -50 -150 z")
	area4 := getArea(vertexList4)
	fmt.Println("vertexList4: ", vertexList4, " lenght: ", length4, " area: ", area4)

	// Checks if vl3 & vl4 intersects
	intersect = isshared.PathIntersect(vertexList3, vertexList4)
	fmt.Println("Is vl3 vl4 Intersect: ", intersect)
}
*/
