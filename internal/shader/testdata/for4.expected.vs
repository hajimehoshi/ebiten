int F0(in int l0);
int F1(void);

int F0(in int l0) {
	return l0;
}

int F1(void) {
	int l0 = 0;
	int l2 = 0;
	l0 = 0;
	for (int l1 = 0; l1 < 10; l1++) {
		int l2 = 0;
		l2 = F0(l1);
		l0 = (l0) + (l2);
	}
	l2 = 0;
	l0 = (l0) + (l2);
	return l0;
}
