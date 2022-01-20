void F0(out vec2 l0);
void F1(out vec2 l0);
void F2(out float l0);
void F3(in float l0);
void F4(in int l0);
void F5(void);

void F0(out vec2 l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	F2(l1);
	l2 = (l1) * (1.0);
	F2(l3);
	l4 = (1.0) * (l3);
	l0 = vec2(l2, l4);
	return;
}

void F1(out vec2 l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	F2(l1);
	l2 = (l1) * (1.0);
	F2(l3);
	l4 = (1.0) * (l3);
	l0 = vec2(l2, l4);
	return;
}

void F2(out float l0) {
	l0 = 1.0;
	return;
}

void F3(in float l0) {
}

void F4(in int l0) {
}

void F5(void) {
	F3(1.0);
	F4(1);
}
