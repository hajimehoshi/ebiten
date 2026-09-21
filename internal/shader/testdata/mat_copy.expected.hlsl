float2x2 F0(in float2x2 l0);
float3x3 F1(in float3x3 l0);
float4x4 F2(in float4x4 l0);
void F3(in float l0, out float2x2 l1, out float3x3 l2, out float4x4 l3);

float2x2 F0(in float2x2 l0) {
	return l0;
}

float3x3 F1(in float3x3 l0) {
	return l0;
}

float4x4 F2(in float4x4 l0) {
	return l0;
}

void F3(in float l0, out float2x2 l1, out float3x3 l2, out float4x4 l3) {
	l1 = float2x2FromScalar(l0);
	l2 = float3x3FromScalar(l0);
	l3 = float4x4FromScalar(l0);
	return;
}
