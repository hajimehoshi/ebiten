float4 F0(in float4 l0);

float4 F0(in float4 l0) {
	float4 l1 = 0.0;
	l1 = mul(l0, float4x4FromScalar(1.0));
	return l1;
}
