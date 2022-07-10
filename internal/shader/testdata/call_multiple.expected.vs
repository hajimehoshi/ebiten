vec2 F0(in vec2 l0);
void F1(in float l0, out float l1, out float l2);

vec2 F0(in vec2 l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	F1((l0).x, l1, l2);
	l3 = l1;
	l4 = l2;
	return vec2(l3, l4);
}

void F1(in float l0, out float l1, out float l2) {
	l1 = l0;
	l2 = l0;
	return;
}
