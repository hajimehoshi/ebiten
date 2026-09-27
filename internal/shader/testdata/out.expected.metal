void F0(bool front_facing, thread float& l0, thread array<float, 4>& l1, thread float4& l2);

void F0(bool front_facing, thread float& l0, thread array<float, 4>& l1, thread float4& l2) {
	l0 = float(0);
	l1[0] = float(0);
	l1[1] = float(0);
	l1[2] = float(0);
	l1[3] = float(0);
	l2 = float4(0);
	return;
}
