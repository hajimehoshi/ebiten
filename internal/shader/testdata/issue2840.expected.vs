float[1] F0(void);
int[1] F1(void);

float[1] F0(void) {
	float l0[1];
	l0[0] = float(0);
	(l0)[0] = 1.0;
	return l0;
}

int[1] F1(void) {
	int l0[1];
	l0[0] = 0;
	(l0)[0] = 1;
	return l0;
}
