void F0(in int l0, out int l1);
void F1(out int l0);

void F0(in int l0, out int l1) {
	l1 = l0;
	return;
}

void F1(out int l0) {
	int l1 = 0;
	int l3 = 0;
	l1 = 0;
	for (int l2 = 0; l2 < 10; l2++) {
		int l3 = 0;
		int l4 = 0;
		F0(l2, l3);
		l4 = l3;
		l1 = (l1) + (l4);
	}
	l3 = 0;
	l1 = (l1) + (l3);
	l0 = l1;
	return;
}
