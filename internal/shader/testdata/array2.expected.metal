array<float2, 3> F0(void);

array<float2, 3> F0(void) {
	array<float2, 2> l0 = {};
	array<float2, 3> l1 = {};
	{
		array<float2, 2> l1 = {};
		l1 = l0;
	}
	return l1;
}
