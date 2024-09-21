void F0(void);

void F0(void) {
	float l0[2];
	l0[0] = float(0);
	l0[1] = float(0);
	float l1[2];
	l1[0] = float(0);
	l1[1] = float(0);
	(l0)[0] = 1.0;
	l1[0] = l0[0];
	l1[1] = l0[1];
}
