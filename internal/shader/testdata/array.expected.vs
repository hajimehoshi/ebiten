uniform vec2 U0[4];

void F0(out vec2 l0[2]);
void F1(out vec2 l0[2]);

void F0(out vec2 l0[2]) {
	vec2 l1[2];
	l1[0] = vec2(0);
	l1[1] = vec2(0);
	l0[0] = l1[0];
	l0[1] = l1[1];
	return;
}

void F1(out vec2 l0[2]) {
	vec2 l1[2];
	l1[0] = vec2(0);
	l1[1] = vec2(0);
	vec2 l2[2];
	l2[0] = vec2(0);
	l2[1] = vec2(0);
	(l1)[0] = vec2(1.0);
	l2[0] = l1[0];
	l2[1] = l1[1];
	((l2)[1]).y = vec2(2.0);
	l0[0] = l2[0];
	l0[1] = l2[1];
	return;
}
