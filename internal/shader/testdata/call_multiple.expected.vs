void F0(in vec2 l0, out vec2 l1) {
	float l2 = 0.0;
	float l3 = 0.0;
	float l4 = 0.0;
	float l5 = 0.0;
	F1((l0).x, (l0).y, l2, l3);
	l4 = l2;
	l5 = l3;
	l1 = vec2(l4, l5);
	return;
}

void F1(in float l0, out float l1, out float l2) {
	l1 = l0;
	l2 = l0;
	return;
}
