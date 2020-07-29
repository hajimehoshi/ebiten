uniform vec2[4] U0;

void F0(out vec2[2] l0);
void F1(out vec2[2] l0);

void F0(out vec2[2] l0) {
	vec2[2] l1 = vec2[2](vec2(0), vec2(0));
	l0 = l1;
	return;
}

void F1(out vec2[2] l0) {
	vec2[2] l1 = vec2[2](vec2(0), vec2(0));
	vec2[2] l2 = vec2[2](vec2(0), vec2(0));
	(l1)[0] = vec2(1.0);
	l2 = l1;
	((l2)[1]).y = vec2(2.0);
	l0 = l2;
	return;
}
