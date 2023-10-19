package game

type dir string

func CheckCollision(A, B Frame) (dir, bool) {
	ax := A.x
	ay := A.y
	aw := A.width
	ah := A.height
	bx := B.x
	by := B.y
	bw := B.width
	bh := B.height
	//left
	if (ax+aw) == bx && ((by < ay && ay < (by+bh)) || (by < (ay+ah) && (ay+ah) < (by+bh))) {
		return "left", true
	}
	//right
	if ax == (bx+bw) && ((by < ay && ay < (by+bh)) || (by < (ay+ah) && (ay+ah) < (by+bh))) {
		return "right", true
	}
	//up
	if (ay+ah) == by && ((bx < ax && ax < (bx+bw)) || (bx < (ax+aw) && (ax+aw) < (bx+bw))) {
		return "up", true
	}
	//down
	if ay == (by+bh) && ((bx < ax && ax < (bx+bw)) || (bx < (ax+aw) && (ax+aw) < (bx+bw))) {
		return "down", true
	}
	return "none", false
}
