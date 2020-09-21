void F0(float l0, float l1, thread bool& l2);

void F0(float l0, float l1, thread bool& l2) {
	float l3 = float(0);
	float l4 = float(0);
	l3 = atan((l1) / (l0));
	l4 = atan2(l1, l0);
	l2 = (l3) == (l4);
	return;
}
