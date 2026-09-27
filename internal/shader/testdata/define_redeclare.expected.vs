vec2 F0(in float l0);
vec2 F1(void);
float F2(in float l0);
void F3(out float l0);
void F4(out float l0, out float l1);

vec2 F0(in float l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	l1 = l0;
	l2 = 1.0;
	l4 = l1;
	l1 = l2;
	l3 = l4;
	return vec2(l1, l3);
}

vec2 F1(void) {
	float l0 = float(0);
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	l0 = 1.0;
	F4(l1, l2);
	l3 = l1;
	l0 = l2;
	return vec2(l0, l3);
}

float F2(in float l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	l1 = 2.0;
	l3 = l0;
	l0 = l1;
	l2 = l3;
	return (l0) + (l2);
}

void F3(out float l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	l0 = float(0);
	l1 = 1.0;
	l3 = 2.0;
	l0 = l1;
	l2 = l3;
	l0 = (l0) + (l2);
	return;
}

void F4(out float l0, out float l1) {
	l0 = 0.0;
	l1 = 0.0;
	return;
}
