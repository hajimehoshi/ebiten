void F0(bool front_facing, thread array<float2, 3>& l0);

void F0(bool front_facing, thread array<float2, 3>& l0) {
	array<float2, 2> l1 = {};
	array<float2, 3> l2 = {};
	{
		array<float2, 2> l2 = {};
		l2 = l1;
	}
	l0 = l2;
	return;
}
