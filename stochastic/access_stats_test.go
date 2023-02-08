package stochastic

import (
	"encoding/json"
	"testing"
)

// TestAccessStatsSimple tests for classifying values based on previous updates
func TestAccessStatsSimple(t *testing.T) {
	// create index accessStat
	accessStat := NewAccessStats[int]()

	const offset = 10
	// place elements into the queue and ensure it overfills
	for i := 0; i < qstatsLen+offset; i++ {
		accessStat.Place(i)
	}

	// check previous element's class
	class := accessStat.Classify(qstatsLen + offset - 1)
	if class != previousEntry {
		t.Fatalf("wrong classification (previous item)")
	}

	// check recent element's class (ones that should be in the queue)
	for i := offset; i < qstatsLen+offset-1; i++ {
		class := accessStat.Classify(i)
		if class != recentEntry {
			t.Fatalf("wrong classification %v (recent data)", i)
		}
	}

	// check class of elements that fell out of the queue
	for i := 1; i < offset; i++ {
		class := accessStat.Classify(i)
		if class != randomEntry {
			t.Fatalf("wrong classification %v (random data)", i)
		}
	}

	// check class of new elements
	class = accessStat.Classify(qstatsLen + offset)
	if class != newEntry {
		t.Fatalf("wrong classification (new data)")
	}

	// check class of new elements
	class = accessStat.Classify(0)
	if class != zeroEntry {
		t.Fatalf("wrong classification (new data)")
	}
}

// TestAccessStatsDistribution tests for JSON output
func TestAccessStatsDistribution(t *testing.T) {
	// create index accessStat
	accessStat := NewAccessStats[int]()

	// place first element
	for i := 0; i < 512; i++ {
		accessStat.Place(i)
		accessStat.Classify(i - 10)
	}

	// produce distribution in JSON format
	jOut, err := json.Marshal(accessStat.NewAccessStatsJSON())
	if err != nil {
		t.Fatalf("Marshalling failed to produce distribution")
	}
	expected := `{"CountingStats":{"n":511,"ecdf":[[0,0],[0.0009784735812133072,0.0019569471624266144],[0.0029354207436399216,0.003913894324853229],[0.004892367906066536,0.005870841487279843],[0.00684931506849315,0.007827788649706457],[0.008806262230919765,0.009784735812133072],[0.010763209393346379,0.011741682974559686],[0.012720156555772993,0.0136986301369863],[0.014677103718199608,0.015655577299412915],[0.016634050880626222,0.01761252446183953],[0.018590998043052837,0.019569471624266144],[0.02054794520547945,0.021526418786692758],[0.022504892367906065,0.023483365949119372],[0.02446183953033268,0.025440313111545987],[0.026418786692759294,0.0273972602739726],[0.02837573385518591,0.029354207436399216],[0.030332681017612523,0.03131115459882583],[0.03228962818003914,0.033268101761252444],[0.03424657534246575,0.03522504892367906],[0.036203522504892366,0.03718199608610567],[0.03816046966731898,0.03913894324853229],[0.040117416829745595,0.0410958904109589],[0.04207436399217221,0.043052837573385516],[0.04403131115459882,0.04500978473581213],[0.04598825831702544,0.046966731898238745],[0.04794520547945205,0.04892367906066536],[0.049902152641878667,0.050880626223091974],[0.05185909980430528,0.05283757338551859],[0.053816046966731895,0.0547945205479452],[0.05577299412915851,0.05675146771037182],[0.057729941291585124,0.05870841487279843],[0.05968688845401174,0.060665362035225046],[0.06164383561643835,0.06262230919765166],[0.06360078277886497,0.06457925636007827],[0.06555772994129158,0.06653620352250489],[0.0675146771037182,0.0684931506849315],[0.06947162426614481,0.07045009784735812],[0.07142857142857142,0.07240704500978473],[0.07338551859099804,0.07436399217221135],[0.07534246575342465,0.07632093933463796],[0.07729941291585127,0.07827788649706457],[0.07925636007827788,0.08023483365949119],[0.0812133072407045,0.0821917808219178],[0.08317025440313111,0.08414872798434442],[0.08512720156555773,0.08610567514677103],[0.08708414872798434,0.08806262230919765],[0.08904109589041095,0.09001956947162426],[0.09099804305283757,0.09197651663405088],[0.09295499021526418,0.09393346379647749],[0.0949119373776908,0.0958904109589041],[0.09686888454011741,0.09784735812133072],[0.09882583170254403,0.09980430528375733],[0.10078277886497064,0.10176125244618395],[0.10273972602739725,0.10371819960861056],[0.10469667318982387,0.10567514677103718],[0.10665362035225048,0.10763209393346379],[0.1086105675146771,0.1095890410958904],[0.11056751467710371,0.11154598825831702],[0.11252446183953033,0.11350293542074363],[0.11448140900195694,0.11545988258317025],[0.11643835616438356,0.11741682974559686],[0.11839530332681017,0.11937377690802348],[0.12035225048923678,0.12133072407045009],[0.1223091976516634,0.1232876712328767],[0.12426614481409001,0.12524461839530332],[0.12622309197651663,0.12720156555772993],[0.12818003913894324,0.12915851272015655],[0.13013698630136986,0.13111545988258316],[0.13209393346379647,0.13307240704500978],[0.13405088062622308,0.1350293542074364],[0.1360078277886497,0.136986301369863],[0.1379647749510763,0.13894324853228962],[0.13992172211350293,0.14090019569471623],[0.14187866927592954,0.14285714285714285],[0.14383561643835616,0.14481409001956946],[0.14579256360078277,0.14677103718199608],[0.14774951076320939,0.1487279843444227],[0.149706457925636,0.1506849315068493],[0.15166340508806261,0.15264187866927592],[0.15362035225048923,0.15459882583170254],[0.15557729941291584,0.15655577299412915],[0.15753424657534246,0.15851272015655576],[0.15949119373776907,0.16046966731898238],[0.16144814090019569,0.162426614481409],[0.1634050880626223,0.1643835616438356],[0.16536203522504891,0.16634050880626222],[0.16731898238747553,0.16829745596868884],[0.16927592954990214,0.17025440313111545],[0.17123287671232876,0.17221135029354206],[0.17318982387475537,0.17416829745596868],[0.175146771037182,0.1761252446183953],[0.1771037181996086,0.1780821917808219],[0.17906066536203522,0.18003913894324852],[0.18101761252446183,0.18199608610567514],[0.18297455968688844,0.18395303326810175],[0.18493150684931506,0.18590998043052837],[0.18688845401174167,0.18786692759295498],[0.1888454011741683,0.1898238747553816],[0.1908023483365949,0.1917808219178082],[0.19275929549902152,0.19373776908023482],[0.19471624266144813,0.19569471624266144],[0.19667318982387474,0.19765166340508805],[0.19863013698630136,0.19960861056751467],[0.20058708414872797,0.20156555772994128],[0.2025440313111546,0.2035225048923679],[0.2045009784735812,0.2054794520547945],[0.20645792563600782,0.20743639921722112],[0.20841487279843443,0.20939334637964774],[0.21037181996086105,0.21135029354207435],[0.21232876712328766,0.21330724070450097],[0.21428571428571427,0.21526418786692758],[0.2162426614481409,0.2172211350293542],[0.2181996086105675,0.2191780821917808],[0.22015655577299412,0.22113502935420742],[0.22211350293542073,0.22309197651663404],[0.22407045009784735,0.22504892367906065],[0.22602739726027396,0.22700587084148727],[0.22798434442270057,0.22896281800391388],[0.2299412915851272,0.2309197651663405],[0.2318982387475538,0.2328767123287671],[0.23385518590998042,0.23483365949119372],[0.23581213307240703,0.23679060665362034],[0.23776908023483365,0.23874755381604695],[0.23972602739726026,0.24070450097847357],[0.24168297455968688,0.24266144814090018],[0.2436399217221135,0.2446183953033268],[0.2455968688845401,0.2465753424657534],[0.24755381604696672,0.24853228962818003],[0.24951076320939333,0.25048923679060664],[0.25146771037182,0.25244618395303325],[0.2534246575342466,0.25440313111545987],[0.2553816046966732,0.2563600782778865],[0.2573385518590998,0.2583170254403131],[0.25929549902152643,0.2602739726027397],[0.26125244618395305,0.2622309197651663],[0.26320939334637966,0.26418786692759294],[0.2651663405088063,0.26614481409001955],[0.2671232876712329,0.26810176125244617],[0.2690802348336595,0.2700587084148728],[0.2710371819960861,0.2720156555772994],[0.27299412915851273,0.273972602739726],[0.27495107632093935,0.2759295499021526],[0.27690802348336596,0.27788649706457924],[0.2788649706457926,0.27984344422700586],[0.2808219178082192,0.28180039138943247],[0.2827788649706458,0.2837573385518591],[0.2847358121330724,0.2857142857142857],[0.28669275929549903,0.2876712328767123],[0.28864970645792565,0.2896281800391389],[0.29060665362035226,0.29158512720156554],[0.2925636007827789,0.29354207436399216],[0.2945205479452055,0.29549902152641877],[0.2964774951076321,0.2974559686888454],[0.2984344422700587,0.299412915851272],[0.30039138943248533,0.3013698630136986],[0.30234833659491195,0.30332681017612523],[0.30430528375733856,0.30528375733855184],[0.3062622309197652,0.30724070450097846],[0.3082191780821918,0.30919765166340507],[0.3101761252446184,0.3111545988258317],[0.312133072407045,0.3131115459882583],[0.31409001956947163,0.3150684931506849],[0.31604696673189825,0.31702544031311153],[0.31800391389432486,0.31898238747553814],[0.3199608610567515,0.32093933463796476],[0.3219178082191781,0.32289628180039137],[0.3238747553816047,0.324853228962818],[0.3258317025440313,0.3268101761252446],[0.32778864970645794,0.3287671232876712],[0.32974559686888455,0.33072407045009783],[0.33170254403131116,0.33268101761252444],[0.3336594911937378,0.33463796477495106],[0.3356164383561644,0.33659491193737767],[0.337573385518591,0.3385518590998043],[0.3395303326810176,0.3405088062622309],[0.34148727984344424,0.3424657534246575],[0.34344422700587085,0.34442270058708413],[0.34540117416829746,0.34637964774951074],[0.3473581213307241,0.34833659491193736],[0.3493150684931507,0.350293542074364],[0.3512720156555773,0.3522504892367906],[0.3532289628180039,0.3542074363992172],[0.35518590998043054,0.3561643835616438],[0.35714285714285715,0.35812133072407043],[0.35909980430528377,0.36007827788649704],[0.3610567514677104,0.36203522504892366],[0.363013698630137,0.3639921722113503],[0.3649706457925636,0.3659491193737769],[0.3669275929549902,0.3679060665362035],[0.36888454011741684,0.3698630136986301],[0.37084148727984345,0.37181996086105673],[0.37279843444227007,0.37377690802348335],[0.3747553816046967,0.37573385518590996],[0.3767123287671233,0.3776908023483366],[0.3786692759295499,0.3796477495107632],[0.3806262230919765,0.3816046966731898],[0.38258317025440314,0.3835616438356164],[0.38454011741682975,0.38551859099804303],[0.38649706457925637,0.38747553816046965],[0.388454011741683,0.38943248532289626],[0.3904109589041096,0.3913894324853229],[0.3923679060665362,0.3933463796477495],[0.3943248532289628,0.3953033268101761],[0.39628180039138944,0.3972602739726027],[0.39823874755381605,0.39921722113502933],[0.40019569471624267,0.40117416829745595],[0.4021526418786693,0.40313111545988256],[0.4041095890410959,0.4050880626223092],[0.4060665362035225,0.4070450097847358],[0.4080234833659491,0.4090019569471624],[0.40998043052837574,0.410958904109589],[0.41193737769080235,0.41291585127201563],[0.41389432485322897,0.41487279843444225],[0.4158512720156556,0.41682974559686886],[0.4178082191780822,0.4187866927592955],[0.4197651663405088,0.4207436399217221],[0.4217221135029354,0.4227005870841487],[0.42367906066536204,0.4246575342465753],[0.42563600782778865,0.42661448140900193],[0.42759295499021527,0.42857142857142855],[0.4295499021526419,0.43052837573385516],[0.4315068493150685,0.4324853228962818],[0.4334637964774951,0.4344422700587084],[0.4354207436399217,0.436399217221135],[0.43737769080234834,0.4383561643835616],[0.43933463796477495,0.44031311154598823],[0.44129158512720157,0.44227005870841485],[0.4432485322896282,0.44422700587084146],[0.4452054794520548,0.4461839530332681],[0.4471624266144814,0.4481409001956947],[0.449119373776908,0.4500978473581213],[0.45107632093933464,0.4520547945205479],[0.45303326810176126,0.45401174168297453],[0.45499021526418787,0.45596868884540115],[0.4569471624266145,0.45792563600782776],[0.4589041095890411,0.4598825831702544],[0.4608610567514677,0.461839530332681],[0.4628180039138943,0.4637964774951076],[0.46477495107632094,0.4657534246575342],[0.46673189823874756,0.46771037181996084],[0.46868884540117417,0.46966731898238745],[0.4706457925636008,0.47162426614481406],[0.4726027397260274,0.4735812133072407],[0.474559686888454,0.4755381604696673],[0.4765166340508806,0.4774951076320939],[0.47847358121330724,0.4794520547945205],[0.48043052837573386,0.48140900195694714],[0.48238747553816047,0.48336594911937375],[0.4843444227005871,0.48532289628180036],[0.4863013698630137,0.487279843444227],[0.4882583170254403,0.4892367906066536],[0.49021526418786693,0.4911937377690802],[0.49217221135029354,0.4931506849315068],[0.49412915851272016,0.49510763209393344],[0.49608610567514677,0.49706457925636005],[0.4980430528375734,0.49902152641878667],[0.5,0.5009784735812133],[0.5019569471624267,0.5029354207436398],[0.5039138943248532,0.5048923679060665],[0.5058708414872799,0.5068493150684932],[0.5078277886497065,0.5088062622309197],[0.5097847358121331,0.5107632093933463],[0.5117416829745597,0.512720156555773],[0.5136986301369864,0.5146771037181996],[0.5156555772994129,0.5166340508806262],[0.5176125244618396,0.5185909980430528],[0.5195694716242661,0.5205479452054794],[0.5215264187866928,0.5225048923679061],[0.5234833659491194,0.5244618395303327],[0.525440313111546,0.5264187866927592],[0.5273972602739726,0.5283757338551859],[0.5293542074363993,0.5303326810176126],[0.5313111545988258,0.5322896281800391],[0.5332681017612525,0.5342465753424657],[0.5352250489236791,0.5362035225048923],[0.5371819960861057,0.538160469667319],[0.5391389432485323,0.5401174168297456],[0.541095890410959,0.5420743639921721],[0.5430528375733855,0.5440313111545988],[0.5450097847358122,0.5459882583170255],[0.5469667318982387,0.547945205479452],[0.5489236790606654,0.5499021526418786],[0.550880626223092,0.5518590998043053],[0.5528375733855186,0.5538160469667319],[0.5547945205479452,0.5557729941291585],[0.5567514677103719,0.557729941291585],[0.5587084148727984,0.5596868884540117],[0.5606653620352251,0.5616438356164384],[0.5626223091976517,0.5636007827788649],[0.5645792563600783,0.5655577299412915],[0.5665362035225049,0.5675146771037182],[0.5684931506849316,0.5694716242661448],[0.5704500978473581,0.5714285714285714],[0.5724070450097848,0.573385518590998],[0.5743639921722113,0.5753424657534246],[0.576320939334638,0.5772994129158513],[0.5782778864970646,0.5792563600782779],[0.5802348336594912,0.5812133072407044],[0.5821917808219178,0.5831702544031311],[0.5841487279843445,0.5851272015655578],[0.586105675146771,0.5870841487279843],[0.5880626223091977,0.5890410958904109],[0.5900195694716243,0.5909980430528375],[0.5919765166340509,0.5929549902152642],[0.5939334637964775,0.5949119373776908],[0.5958904109589042,0.5968688845401173],[0.5978473581213307,0.598825831702544],[0.5998043052837574,0.6007827788649707],[0.601761252446184,0.6027397260273972],[0.6037181996086106,0.6046966731898238],[0.6056751467710372,0.6066536203522505],[0.6076320939334638,0.6086105675146771],[0.6095890410958904,0.6105675146771037],[0.6115459882583171,0.6125244618395302],[0.6135029354207436,0.6144814090019569],[0.6154598825831703,0.6164383561643836],[0.6174168297455969,0.6183953033268101],[0.6193737769080235,0.6203522504892367],[0.6213307240704501,0.6223091976516634],[0.6232876712328768,0.62426614481409],[0.6252446183953033,0.6262230919765166],[0.62720156555773,0.6281800391389432],[0.6291585127201565,0.6301369863013698],[0.6311154598825832,0.6320939334637965],[0.6330724070450098,0.6340508806262231],[0.6350293542074364,0.6360078277886496],[0.636986301369863,0.6379647749510763],[0.6389432485322897,0.639921722113503],[0.6409001956947162,0.6418786692759295],[0.6428571428571429,0.6438356164383561],[0.6448140900195695,0.6457925636007827],[0.6467710371819961,0.6477495107632094],[0.6487279843444227,0.649706457925636],[0.6506849315068494,0.6516634050880625],[0.6526418786692759,0.6536203522504892],[0.6545988258317026,0.6555772994129159],[0.6565557729941291,0.6575342465753424],[0.6585127201565558,0.659491193737769],[0.6604696673189824,0.6614481409001957],[0.662426614481409,0.6634050880626223],[0.6643835616438356,0.6653620352250489],[0.6663405088062623,0.6673189823874754],[0.6682974559686888,0.6692759295499021],[0.6702544031311155,0.6712328767123288],[0.6722113502935421,0.6731898238747553],[0.6741682974559687,0.6751467710371819],[0.6761252446183953,0.6771037181996086],[0.678082191780822,0.6790606653620352],[0.6800391389432485,0.6810176125244618],[0.6819960861056752,0.6829745596868884],[0.6839530332681018,0.684931506849315],[0.6859099804305284,0.6868884540117417],[0.687866927592955,0.6888454011741683],[0.6898238747553816,0.6908023483365948],[0.6917808219178082,0.6927592954990215],[0.6937377690802349,0.6947162426614482],[0.6956947162426614,0.6966731898238747],[0.6976516634050881,0.6986301369863013],[0.6996086105675147,0.700587084148728],[0.7015655577299413,0.7025440313111546],[0.7035225048923679,0.7045009784735812],[0.7054794520547946,0.7064579256360077],[0.7074363992172211,0.7084148727984344],[0.7093933463796478,0.7103718199608611],[0.7113502935420744,0.7123287671232876],[0.713307240704501,0.7142857142857142],[0.7152641878669276,0.7162426614481409],[0.7172211350293543,0.7181996086105675],[0.7191780821917808,0.7201565557729941],[0.7211350293542075,0.7221135029354206],[0.723091976516634,0.7240704500978473],[0.7250489236790607,0.726027397260274],[0.7270058708414873,0.7279843444227005],[0.7289628180039139,0.7299412915851271],[0.7309197651663405,0.7318982387475538],[0.7328767123287672,0.7338551859099804],[0.7348336594911937,0.735812133072407],[0.7367906066536204,0.7377690802348336],[0.738747553816047,0.7397260273972602],[0.7407045009784736,0.7416829745596869],[0.7426614481409002,0.7436399217221135],[0.7446183953033269,0.74559686888454],[0.7465753424657534,0.7475538160469667],[0.7485322896281801,0.7495107632093934],[0.7504892367906066,0.7514677103718199],[0.7524461839530333,0.7534246575342465],[0.7544031311154599,0.7553816046966731],[0.7563600782778865,0.7573385518590998],[0.7583170254403131,0.7592954990215264],[0.7602739726027398,0.7612524461839529],[0.7622309197651663,0.7632093933463796],[0.764187866927593,0.7651663405088063],[0.7661448140900196,0.7671232876712328],[0.7681017612524462,0.7690802348336594],[0.7700587084148728,0.7710371819960861],[0.7720156555772995,0.7729941291585127],[0.773972602739726,0.7749510763209393],[0.7759295499021527,0.7769080234833659],[0.7778864970645792,0.7788649706457925],[0.7798434442270059,0.7808219178082192],[0.7818003913894325,0.7827788649706457],[0.7837573385518591,0.7847358121330723],[0.7857142857142857,0.786692759295499],[0.7876712328767124,0.7886497064579256],[0.7896281800391389,0.7906066536203522],[0.7915851272015656,0.7925636007827788],[0.7935420743639922,0.7945205479452054],[0.7954990215264188,0.7964774951076321],[0.7974559686888454,0.7984344422700587],[0.799412915851272,0.8003913894324852],[0.8013698630136986,0.8023483365949119],[0.8033268101761253,0.8043052837573386],[0.8052837573385518,0.8062622309197651],[0.8072407045009785,0.8082191780821917],[0.8091976516634051,0.8101761252446184],[0.8111545988258317,0.812133072407045],[0.8131115459882583,0.8140900195694716],[0.815068493150685,0.8160469667318981],[0.8170254403131115,0.8180039138943248],[0.8189823874755382,0.8199608610567515],[0.8209393346379648,0.821917808219178],[0.8228962818003914,0.8238747553816046],[0.824853228962818,0.8258317025440313],[0.8268101761252447,0.8277886497064579],[0.8287671232876712,0.8297455968688845],[0.8307240704500979,0.831702544031311],[0.8326810176125244,0.8336594911937377],[0.8346379647749511,0.8356164383561644],[0.8365949119373777,0.837573385518591],[0.8385518590998043,0.8395303326810175],[0.8405088062622309,0.8414872798434442],[0.8424657534246576,0.8434442270058709],[0.8444227005870841,0.8454011741682974],[0.8463796477495108,0.847358121330724],[0.8483365949119374,0.8493150684931506],[0.850293542074364,0.8512720156555773],[0.8522504892367906,0.8532289628180039],[0.8542074363992173,0.8551859099804304],[0.8561643835616438,0.8571428571428571],[0.8581213307240705,0.8590998043052838],[0.860078277886497,0.8610567514677103],[0.8620352250489237,0.8630136986301369],[0.8639921722113503,0.8649706457925636],[0.8659491193737769,0.8669275929549902],[0.8679060665362035,0.8688845401174168],[0.8698630136986302,0.8708414872798433],[0.8718199608610567,0.87279843444227],[0.8737769080234834,0.8747553816046967],[0.87573385518591,0.8767123287671232],[0.8776908023483366,0.8786692759295498],[0.8796477495107632,0.8806262230919765],[0.8816046966731899,0.8825831702544031],[0.8835616438356164,0.8845401174168297],[0.8855185909980431,0.8864970645792563],[0.8874755381604696,0.8884540117416829],[0.8894324853228963,0.8904109589041096],[0.8913894324853229,0.8923679060665362],[0.8933463796477495,0.8943248532289627],[0.8953033268101761,0.8962818003913894],[0.8972602739726028,0.898238747553816],[0.8992172211350293,0.9001956947162426],[0.901174168297456,0.9021526418786692],[0.9031311154598826,0.9041095890410958],[0.9050880626223092,0.9060665362035225],[0.9070450097847358,0.9080234833659491],[0.9090019569471625,0.9099804305283756],[0.910958904109589,0.9119373776908023],[0.9129158512720157,0.913894324853229],[0.9148727984344422,0.9158512720156555],[0.9168297455968689,0.9178082191780821],[0.9187866927592955,0.9197651663405088],[0.9207436399217221,0.9217221135029354],[0.9227005870841487,0.923679060665362],[0.9246575342465754,0.9256360078277885],[0.9266144814090019,0.9275929549902152],[0.9285714285714286,0.9295499021526419],[0.9305283757338552,0.9315068493150684],[0.9324853228962818,0.933463796477495],[0.9344422700587084,0.9354207436399217],[0.9363992172211351,0.9373776908023483],[0.9383561643835616,0.9393346379647749],[0.9403131115459883,0.9412915851272015],[0.9422700587084148,0.9432485322896281],[0.9442270058708415,0.9452054794520548],[0.9461839530332681,0.9471624266144814],[0.9481409001956947,0.9491193737769079],[0.9500978473581213,0.9510763209393346],[0.952054794520548,0.9530332681017613],[0.9540117416829745,0.9549902152641878],[0.9559686888454012,0.9569471624266144],[0.9579256360078278,0.958904109589041],[0.9598825831702544,0.9608610567514677],[0.961839530332681,0.9628180039138943],[0.9637964774951077,0.9647749510763208],[0.9657534246575342,0.9667318982387475],[0.9677103718199609,0.9686888454011742],[0.9696673189823874,0.9706457925636007],[0.9716242661448141,0.9726027397260273],[0.9735812133072407,0.974559686888454],[0.9755381604696673,0.9765166340508806],[0.9774951076320939,0.9784735812133072],[0.9794520547945206,0.9804305283757337],[0.9814090019569471,0.9823874755381604],[0.9833659491193738,0.9843444227005871],[0.9853228962818004,0.9863013698630136],[0.987279843444227,0.9882583170254402],[0.9892367906066536,0.9902152641878669],[0.9911937377690803,0.9921722113502935],[0.9931506849315068,0.9941291585127201],[0.9951076320939335,0.9960861056751467],[0.99706457925636,0.9980430528375733],[0.9990215264187867,1],[1,1]]},"QueuingStats":{"distribution":[0,0,0,0,0,0,0,0,0,0,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}}`
	if string(jOut) != expected {
		t.Fatalf("produced wrong JSON output %v", string(jOut))
	}
}
