int F0(in int l0);
float F1(in float l0);
int F2(void);
float F3(void);

int F0(in int l0) {
	return (1) + (l0);
}

float F1(in float l0) {
	return (1.0) + (l0);
}

int F2(void) {
	int l0 = 0;
	l0 = F0(1);
	return l0;
}

float F3(void) {
	float l0 = float(0);
	l0 = F1(1.0);
	return l0;
}
