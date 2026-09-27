void F0(bool front_facing, thread array<float2, 3>& l0);

void F0(bool front_facing, thread array<float2, 3>& l0) {
	array<float2, 2> l1;
	l1[0] = float2(0);
	l1[1] = float2(0);
	array<float2, 3> l2;
	l2[0] = float2(0);
	l2[1] = float2(0);
	l2[2] = float2(0);
	{
		array<float2, 2> l2;
		l2[0] = float2(0);
		l2[1] = float2(0);
		l2[0] = l1[0];
		l2[1] = l1[1];
	}
	l0[0] = l2[0];
	l0[1] = l2[1];
	l0[2] = l2[2];
	return;
}
