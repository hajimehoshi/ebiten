// https://github.com/technosaurus/PDMP3
// License: Public Domain

#include "pdmp3.h"

#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <unistd.h>
#include <stdlib.h>

#define C_EOF              0xffffffff
#define C_PI                   3.14159265358979323846
#define C_INV_SQRT_2           0.70710678118654752440
#define Hz                           1
#define kHz                    1000*Hz
#define bit_s                        1
#define kbit_s                 1000*bit_s
#define FRAG_SIZE_LN2     0x0011 /* 2^17=128kb */
#define FRAG_NUMS         0x0004

#define DBG(str,args...) { printf(str,## args); printf("\n"); }
#define ERR(str,args...) { fprintf(stderr,str,## args) ; fprintf(stderr,"\n"); }
#define EXIT(str,args...) { printf(str,## args);  printf("\n"); exit(0); }

#ifdef DEBUG //debug functions
static void dmp_fr(t_mpeg1_header *hdr);
static void dmp_si(t_mpeg1_header *hdr,t_mpeg1_side_info *si);
static void dmp_scf(t_mpeg1_side_info *si,t_mpeg1_main_data *md,int gr,int ch);
static void dmp_huff(t_mpeg1_main_data *md,int gr,int ch);
static void dmp_samples(t_mpeg1_main_data *md,int gr,int ch,int type);
#else
#define dmp_fr(...) do{}while(0)
#define dmp_si(...) do{}while(0)
#define dmp_scf(...) do{}while(0)
#define dmp_huff(...) do{}while(0)
#define dmp_samples(...) do{}while(0)
#endif

static void audio_write(unsigned *samples,unsigned nsamples,int sample_rate);
static void audio_write_raw(unsigned *samples,unsigned nsamples);
static void Decode_L3_Init_Song(void);
static void Error(const char *s,int e);
static void Read_Ancillary(void);

static const unsigned g_mpeg1_bitrates[3 /* layer 1-3 */][15 /* header bitrate_index */] = {
  {    /* Layer 1 */
         0, 32000, 64000, 96000,128000,160000,192000,224000,
    256000,288000,320000,352000,384000,416000,448000
  },{  /* Layer 2 */
         0, 32000, 48000, 56000, 64000, 80000, 96000,112000,
    128000,160000,192000,224000,256000,320000,384000
  },{   /* Layer 3 */
         0, 32000, 40000, 48000, 56000, 64000, 80000, 96000,
    112000,128000,160000,192000,224000,256000,320000
  }
},
g_sampling_frequency[3] = { 44100 * Hz,48000 * Hz,32000 * Hz };

unsigned synth_init = 1;

/* Scale factor band indices
 *
 * One table per sample rate. Each table contains the frequency indices
 * for the 12 short and 21 long scalefactor bands. The short indices
 * must be multiplied by 3 to get the actual index.
 */
static const t_sf_band_indices g_sf_band_indices[3 /* Sampling freq. */] = {
    {
      {0,4,8,12,16,20,24,30,36,44,52,62,74,90,110,134,162,196,238,288,342,418,576},
      {0,4,8,12,16,22,30,40,52,66,84,106,136,192}
    },
    {
      {0,4,8,12,16,20,24,30,36,42,50,60,72,88,106,128,156,190,230,276,330,384,576},
      {0,4,8,12,16,22,28,38,50,64,80,100,126,192}
    },
    {
      {0,4,8,12,16,20,24,30,36,44,54,66,82,102,126,156,194,240,296,364,448,550,576},
      {0,4,8,12,16,22,30,42,58,78,104,138,180,192}
    }
  };

t_mpeg1_header    g_frame_header;
t_mpeg1_side_info g_side_info;  /* < 100 words */
t_mpeg1_main_data g_main_data;  /* Large static data(~2500 words) */

#ifdef DEBUG
static void dmp_fr(t_mpeg1_header *hdr){
  printf("rate %d,sfreq %d,pad %d,mod %d,modext %d,emph %d\n",
          hdr->bitrate_index,hdr->sampling_frequency,hdr->padding_bit,
          hdr->mode,hdr->mode_extension,hdr->emphasis);
}

static void dmp_si(t_mpeg1_header *hdr,t_mpeg1_side_info *si){
  int nch,ch,gr;

  nch = hdr->mode == mpeg1_mode_single_channel ? 1 : 2;
  printf("main_data_begin %d,priv_bits %d\n",si->main_data_begin,si->private_bits);
  for(ch = 0; ch < nch; ch++) {
    printf("scfsi %d %d %d %d\n",si->scfsi[ch][0],si->scfsi[ch][1],si->scfsi[ch][2],si->scfsi[ch][3]);
    for(gr = 0; gr < 2; gr++) {
      printf("p23l %d,bv %d,gg %d,scfc %d,wsf %d,bt %d\n",
              si->part2_3_length[gr][ch],si->big_values[gr][ch],
              si->global_gain[gr][ch],si->scalefac_compress[gr][ch],
              si->win_switch_flag[gr][ch],si->block_type[gr][ch]);
      if(si->win_switch_flag[gr][ch]) {
        printf("mbf %d,ts1 %d,ts2 %d,sbg1 %d,sbg2 %d,sbg3 %d\n",
                si->mixed_block_flag[gr][ch],si->table_select[gr][ch][0],
                si->table_select[gr][ch][1],si->subblock_gain[gr][ch][0],
                si->subblock_gain[gr][ch][1],si->subblock_gain[gr][ch][2]);
      }else{
        printf("ts1 %d,ts2 %d,ts3 %d\n",si->table_select[gr][ch][0],
                si->table_select[gr][ch][1],si->table_select[gr][ch][2]);
      }
      printf("r0c %d,r1c %d\n",si->region0_count[gr][ch],si->region1_count[gr][ch]);
      printf("pf %d,scfs %d,c1ts %d\n",si->preflag[gr][ch],si->scalefac_scale[gr][ch],si->count1table_select[gr][ch]);
    }
  }
}

static void dmp_scf(t_mpeg1_side_info *si,t_mpeg1_main_data *md,int gr,int ch){
  int sfb,win;

  if((si->win_switch_flag[gr][ch] != 0) &&(si->block_type[gr][ch] == 2)) {
    if(si->mixed_block_flag[gr][ch] != 0) { /* First the long block scalefacs */
      for(sfb = 0; sfb < 8; sfb++)
        printf("scfl%d %d%s",sfb,md->scalefac_l[gr][ch][sfb],(sfb==7)?"\n":",");
      for(sfb = 3; sfb < 12; sfb++) /* And next the short block scalefacs */
        for(win = 0; win < 3; win++)
          printf("scfs%d,%d %d%s",sfb,win,md->scalefac_s[gr][ch][sfb][win],(win==2)?"\n":",");
    }else{                /* Just short blocks */
      for(sfb = 0; sfb < 12; sfb++)
        for(win = 0; win < 3; win++)
          printf("scfs%d,%d %d%s",sfb,win,md->scalefac_s[gr][ch][sfb][win],(win==2)?"\n":",");
    }
  }else for(sfb = 0; sfb < 21; sfb++) /* Just long blocks; scalefacs first */
          printf("scfl%d %d%s",sfb,md->scalefac_l[gr][ch][sfb], (sfb == 20)?"\n":",");
}

static void dmp_huff(t_mpeg1_main_data *md,int gr,int ch){
  int i;
  printf("HUFFMAN\n");
  for(i = 0; i < 576; i++) printf("%d: %d\n",i,(int) md->is[gr][ch][i]);
}

static void dmp_samples(t_mpeg1_main_data *md,int gr,int ch,int type){
  int i,val;
  extern double rint(double);

  printf("SAMPLES%d\n",type);
  for(i = 0; i < 576; i++) {
    val =(int) rint(md->is[gr][ch][i] * 32768.0);
    if(val >= 32768) val = 32767;
    if(val < -32768) val = -32768;
    printf("%d: %d\n",i,val);
  }
}
#endif

/**Description: decodes a layer 3 bitstream into audio samples.
* Parameters: Outdata vector.
* Return value: OK or ERROR if the frame contains errors.
* Author: Krister Lagerström(krister@kmlager.com) **/
int Decode_L3(void){
  unsigned gr,ch,nch,out[576];

  /* Number of channels(1 for mono and 2 for stereo) */
  nch =(g_frame_header.mode == mpeg1_mode_single_channel ? 1 : 2);
  for(gr = 0; gr < 2; gr++) {
    for(ch = 0; ch < nch; ch++) {
      dmp_scf(&g_side_info,&g_main_data,gr,ch); //noop unless debug
      dmp_huff(&g_main_data,gr,ch); //noop unless debug
      L3_Requantize(gr,ch); /* Requantize samples */
      dmp_samples(&g_main_data,gr,ch,0); //noop unless debug
      L3_Reorder(gr,ch); /* Reorder short blocks */
    } /* end for(ch... */
    L3_Stereo(gr); /* Stereo processing */
    dmp_samples(&g_main_data,gr,0,1); //noop unless debug
    dmp_samples(&g_main_data,gr,1,1); //noop unless debug
    for(ch = 0; ch < nch; ch++) {
      L3_Antialias(gr,ch); /* Antialias */
      dmp_samples(&g_main_data,gr,ch,2); //noop unless debug
      L3_Hybrid_Synthesis(gr,ch); /*(IMDCT,windowing,overlapp add) */
      L3_Frequency_Inversion(gr,ch); /* Frequency inversion */
     dmp_samples(&g_main_data,gr,ch,3); //noop unless debug
      L3_Subband_Synthesis(gr,ch,out); /* Polyphase subband synthesis */
    } /* end for(ch... */
#ifdef DEBUG
    {
      int i,ctr = 0;
      printf("PCM:\n");
      for(i = 0; i < 576; i++) {
        printf("%d: %d\n",ctr++,(out[i] >> 16) & 0xffff);
        if(nch == 2) printf("%d: %d\n",ctr++,out[i] & 0xffff);
      }
    }
#endif /* DEBUG */
     /*FIXME - replace with simple interface stream*/
    audio_write((unsigned *) out,576,
                 g_sampling_frequency[g_frame_header.sampling_frequency]);
  } /* end for(gr... */
  return(OK);   /* Done */
}

/**Description: Search for next frame and read it into  buffer. Main data in
   this frame is saved for two frames since it might be needed by them.
* Parameters: None
* Return value: OK if a frame is successfully read,ERROR otherwise.
* Author: Krister Lagerström(krister@kmlager.com) **/
int Read_Frame(void){
  unsigned first = 0;

  if(Get_Filepos() == 0) Decode_L3_Init_Song();
  /* Try to find the next frame in the bitstream and decode it */
  if(Read_Header() != OK) return(ERROR);
#ifdef DEBUG
  { static int framenum = 0;
    printf("\nFrame %d\n",framenum++);
    dmp_fr(&g_frame_header);
  }
    DBG("Starting decode,Layer: %d,Rate: %6d,Sfreq: %05d",
         g_frame_header.layer,
         g_mpeg1_bitrates[g_frame_header.layer - 1][g_frame_header.bitrate_index],
         g_sampling_frequency[g_frame_header.sampling_frequency]);
#endif
  /* Get CRC word if present */
  if((g_frame_header.protection_bit==0)&&(Read_CRC()!=OK)) return(ERROR);
  if(g_frame_header.layer == 3) {  /* Get audio data */
    Read_Audio_L3();  /* Get side info */
    dmp_si(&g_frame_header,&g_side_info); /* DEBUG */
    /* If there's not enough main data in the bit reservoir,
     * signal to calling function so that decoding isn't done! */
    /* Get main data(scalefactors and Huffman coded frequency data) */
    if(Read_Main_L3() != OK) return(ERROR);
  }else{
    ERR("Only layer 3(!= %d) is supported!\n",g_frame_header.layer);
    return(ERROR);
  }
  return(OK);
}

/**Description: TBD
* Parameters: TBD
* Return value: TBD
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Error(const char *s,int e){
  (void) fwrite(s,1,strlen(s),stderr);
  exit(e);
}

/**Description: reinit decoder before playing new song,or seeking current song.
* Parameters: None
* Return value: None
* Author: Krister Lagerström(krister@kmlager.com) **/
static void Decode_L3_Init_Song(void){
  synth_init = 1;
}

/**Description: output audio data
* Parameters: Pointers to the samples,the number of samples
* Return value: None
* Author: Krister Lagerström(krister@kmlager.com) **/
static void audio_write(unsigned *samples,unsigned nsamples,int sample_rate){
  static int init = 0,audio,curr_sample_rate = 0;
  int tmp,dsp_speed = 44100,dsp_stereo = 2;

#ifdef OUTPUT_RAW
  audio_write_raw(samples,nsamples);
#endif /* OUTPUT_RAW */
  return;
} /* audio_write() */

/******************************************************************************
*
* Name: audio_write_raw
* Author: Krister Lagerström(krister@unidata.se)
* Description: This function is used to output raw data
* Parameters: Pointers to the samples,the number of samples
* Return value: None
* Revision History:
* Author   Date    Change
* krister  010101  Initial revision
*
******************************************************************************/
static void audio_write_raw(unsigned *samples,unsigned nsamples){
  char fname[1024];
  unsigned lo,hi;
  int i,nch;
  unsigned short s[576*2];

  nch =(g_frame_header.mode == mpeg1_mode_single_channel ? 1 : 2);
  for(i = 0; i < nsamples; i++) {
    if(nch == 1) {
      lo = samples[i] & 0xffff;
      s[i] = lo;
    }else{
      lo = samples[i] & 0xffff;
      hi =(samples[i] & 0xffff0000) >> 16;
      s[2*i] = hi;
      s[2*i+1] = lo;
    }
  }
  if(writeToWriter((char *) s,nsamples * 2 * nch) != nsamples * 2 * nch)
    Error("Unable to write raw data\n",-1);
  return;
} /* audio_write_raw() */
