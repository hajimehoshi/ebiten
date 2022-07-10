vec4 F0(in vec2 l0);
void F1(out float l0, out float l1);

vec4 F0(in vec2 l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	F1(l1, l2);
	l3 = l1;
	l4 = l2;
	return vec4(l0, l3, l3);
}

void F1(out float l0, out float l1) {
	l0 = 0.0;
	l1 = 0.0;
	return;
}
