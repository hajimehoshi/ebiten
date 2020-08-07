void F0(in vec2 l0, out vec4 l1);
void F1(out float l0, out float l1);

void F0(in vec2 l0, out vec4 l1) {
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	float l5 = float(0);
	F1(l4, l5);
	l2 = l4;
	l3 = l5;
	l1 = vec4(l0, l2, l3);
	return;
}

void F1(out float l0, out float l1) {
	l0 = 0.0;
	l1 = 0.0;
	return;
}
