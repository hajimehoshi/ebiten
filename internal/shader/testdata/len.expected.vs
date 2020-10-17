void F0(in vec2 l0, out int l1);
void F1(in vec2 l0, out int l1);

void F0(in vec2 l0, out int l1) {
	int l2[2];
	l2[0] = 0;
	l2[1] = 0;
	l1 = 2;
	return;
}

void F1(in vec2 l0, out int l1) {
	int l2[3];
	l2[0] = 0;
	l2[1] = 0;
	l2[2] = 0;
	l1 = 3;
	return;
}
