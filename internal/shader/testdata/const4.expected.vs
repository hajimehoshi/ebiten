void F0(in int l0, out int l1);
void F1(in float l0, out float l1);
void F2(out int l0);
void F3(out float l0);

void F0(in int l0, out int l1) {
	l1 = (1) + (l0);
	return;
}

void F1(in float l0, out float l1) {
	l1 = (1.0) + (l0);
	return;
}

void F2(out int l0) {
	int l1 = 0;
	F0(1, l1);
	l0 = l1;
	return;
}

void F3(out float l0) {
	float l1 = float(0);
	F1(1.0, l1);
	l0 = l1;
	return;
}
