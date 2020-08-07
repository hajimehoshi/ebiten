void F0(in vec2 l0);
void F1(in float l0, in float l1, out float l2, out float l3);

void F0(in vec2 l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	F1((l0).x, (l0).y, l1, l2);
	F1(l1, l2, l3, l4);
}

void F1(in float l0, in float l1, out float l2, out float l3) {
	l2 = l0;
	l3 = l1;
	return;
}
