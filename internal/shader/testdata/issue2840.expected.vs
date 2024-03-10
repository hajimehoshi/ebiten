void F0(out float l0[1]);
void F1(out int l0[1]);

void F0(out float l0[1]) {
	float l1[1];
	l1[0] = float(0);
	(l1)[0] = 1.0;
	l0[0] = l1[0];
	return;
}

void F1(out int l0[1]) {
	int l1[1];
	l1[0] = 0;
	(l1)[0] = 1;
	l0[0] = l1[0];
	return;
}
