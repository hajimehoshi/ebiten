void F0(out float l0, out float l1[4], out vec4 l2);
void F1(out float l0, out float l1[4], out vec4 l2);

void F0(out float l0, out float l1[4], out vec4 l2) {
	l0 = float(0);
	l1[0] = float(0);
	l1[1] = float(0);
	l1[2] = float(0);
	l1[3] = float(0);
	l2 = vec4(0);
	return;
}

void F1(out float l0, out float l1[4], out vec4 l2) {
	float l3 = float(0);
	float l4[4];
	l4[0] = float(0);
	l4[1] = float(0);
	l4[2] = float(0);
	l4[3] = float(0);
	vec4 l5 = vec4(0);
	l0 = float(0);
	l1[0] = float(0);
	l1[1] = float(0);
	l1[2] = float(0);
	l1[3] = float(0);
	l2 = vec4(0);
	F0(l3, l4, l5);
	l0 = l3;
	l1[0] = l4[0];
	l1[1] = l4[1];
	l1[2] = l4[2];
	l1[3] = l4[3];
	l2 = l5;
	return;
}
