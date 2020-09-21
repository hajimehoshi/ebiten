void F0(in float l0, in float l1, out bool l2);

void F0(in float l0, in float l1, out bool l2) {
	float l3 = float(0);
	float l4 = float(0);
	l3 = atan((l1) / (l0));
	l4 = atan(l1, l0);
	l2 = (l3) == (l4);
	return;
}
