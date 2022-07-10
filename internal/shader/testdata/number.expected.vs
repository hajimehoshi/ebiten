vec2 F0(void);
vec2 F1(void);
float F2(void);
void F3(in float l0);
void F4(in int l0);
void F5(void);

vec2 F0(void) {
	float l0 = float(0);
	float l1 = float(0);
	l0 = (F2()) * (1.0);
	l1 = (1.0) * (F2());
	return vec2(l0, l1);
}

vec2 F1(void) {
	float l0 = float(0);
	float l1 = float(0);
	l0 = (F2()) * (1.0);
	l1 = (1.0) * (F2());
	return vec2(l0, l1);
}

float F2(void) {
	return 1.0;
}

void F3(in float l0) {
}

void F4(in int l0) {
}

void F5(void) {
	F3(1.0);
	F4(1);
}
