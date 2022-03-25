void F0(in float4 l0, out float4 l1) {
	float4 l2 = 0.0;
	l2 = mul(l0, float4x4FromScalar(1.0));
	l1 = l2;
	return;
}
