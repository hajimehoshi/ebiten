void F0(out vec2 l0);
void F1(out vec2 l0);
void F2(out float l0);
void F3(out int l0);
void F4(in float l0);
void F5(in int l0);
void F6(void);

void F0(out vec2 l0) {
	float l1 = float(0);
	int l2 = 0;
	float l3 = float(0);
	int l4 = 0;
	float l5 = float(0);
	float l6 = float(0);
	F2(l1);
	F3(l2);
	l3 = (l1) * (l2);
	F3(l4);
	F2(l5);
	l6 = (l4) * (l5);
	l0 = vec2(l3, l6);
	return;
}

void F1(out vec2 l0) {
	float l1 = float(0);
	int l2 = 0;
	float l3 = float(0);
	int l4 = 0;
	float l5 = float(0);
	float l6 = float(0);
	F2(l1);
	F3(l2);
	l3 = (l1) * (l2);
	F3(l4);
	F2(l5);
	l6 = (l4) * (l5);
	l0 = vec2(l3, l6);
	return;
}

void F2(out float l0) {
	l0 = 1.0;
	return;
}

void F3(out int l0) {
	l0 = 1;
	return;
}

void F4(in float l0) {
}

void F5(in int l0) {
}

void F6(void) {
	F4(1.0);
	F5(1);
}
