void F0(out int l0, out int l1);
void F1(void);

void F0(out int l0, out int l1) {
	l0 = 1;
	l1 = 1;
	return;
}

void F1(void) {
	int l0 = 0;
	int l1 = 0;
	int l2 = 0;
	int l3 = 0;
	int l4 = 0;
	int l5 = 0;
	int l6 = 0;
	int l7 = 0;
	int l8 = 0;
	int l9 = 0;
	F0(l0, l1);
	F0(l2, l3);
	l4 = l2;
	F0(l6, l7);
	l9 = l7;
}
