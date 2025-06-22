bool F0(bool front_facing, float l0, float l1);

bool F0(bool front_facing, float l0, float l1) {
	float l2 = float(0);
	float l3 = float(0);
	l2 = atan((l1) / (l0));
	l3 = atan2(l1, l0);
	return (l2) == (l3);
}
