void F0(out vec2 l0);

void F0(out vec2 l0) {
	bool l1 = false;
	l1 = true;
	if (l1) {
		l0 = vec2(0.0);
		return;
	} else {
		l0 = vec2(1.0);
		return;
	}
}
