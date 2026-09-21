float2x2 F0(bool front_facing, float2x2 l0);
float3x3 F1(bool front_facing, float3x3 l0);
float4x4 F2(bool front_facing, float4x4 l0);
void F3(bool front_facing, float l0, thread float2x2& l1, thread float3x3& l2, thread float4x4& l3);

float2x2 F0(bool front_facing, float2x2 l0) {
	return l0;
}

float3x3 F1(bool front_facing, float3x3 l0) {
	return l0;
}

float4x4 F2(bool front_facing, float4x4 l0) {
	return l0;
}

void F3(bool front_facing, float l0, thread float2x2& l1, thread float3x3& l2, thread float4x4& l3) {
	l1 = float2x2(l0);
	l2 = float3x3(l0);
	l3 = float4x4(l0);
	return;
}
