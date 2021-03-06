// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
package vm

import (
	"errors"
	"github.com/hashicorp/golang-lru"
	"github.com/taiyuechain/taiyuechain/common"
	"github.com/taiyuechain/taiyuechain/crypto"
	"fmt"
)

////cache

var whitelistIsWork_SendTx bool
var whitelistIsWork_CrtContract bool

var(
	NotRootSendTxError = errors.New("not root sendtx error")
	FristGreateGropError = errors.New("the frist create grop err create not equl from")
	MemberAreadInGropError = errors.New("member alread in grop")
	MemberNotInGropError = errors.New("member not in grop")
	MemberNotSendTxPermError = errors.New("member not send tx permission")
	ErrorMemberAlreadIn = errors.New("Mamber alread have this perminssion")
	GropNameAlreadyUseError = errors.New("Grop Name alread use")
	GropNotExitError = errors.New("Grop not exit")
	MemberGropNotExitError = errors.New("member Grop not exit")
	ContractAlreadyCreatePremError = errors.New("Contract already create prem")
	ContractNotCreatePremError = errors.New("Contract not create prem")
	ContractPremFlagError = errors.New("Contract premission flage error")
	MemberInBlackListError = errors.New("member is in black list")
	MemberNotSentTXPerm = errors.New("member have not sentTx permission")
)

var PerminCache *PerminssionCache

type ModifyPerminType int

const (
	ModifyPerminType_Nil ModifyPerminType = iota
	ModifyPerminType_AddSendTxPerm //add send tx permission
	ModifyPerminType_DelSendTxPerm //del send tx permission
	ModifyPerminType_AddSendTxManagerPerm
	ModifyPerminType_DelSendTxManagerPerm
	ModifyPerminType_AddCrtContractPerm
	ModifyPerminType_DelCrtContractPerm
	ModifyPerminType_AddCrtContractManagerPerm
	ModifyPerminType_DelCrtContractManagerPerm
	ModifyPerminType_AddGropManagerPerm
	ModifyPerminType_DelGropManagerPerm
	ModifyPerminType_AddGropMemberPerm
	ModifyPerminType_DelGropMemberPerm
	ModifyPerminType_CrtContractPerm //create permission for contract
	ModifyPerminType_AddContractMemberPerm
	ModifyPerminType_DelContractMemberPerm
	ModifyPerminType_AddContractManagerPerm
	ModifyPerminType_DelContractManagerPerm
	ModifyPerminType_AddWhitListPerm
	ModifyPerminType_DelWhitListPerm
	ModifyPerminType_AddBlockListPerm
	ModifyPerminType_DelBlockListPerm
	PerminType_SendTx
	PerminType_CreateContract // this is memeber owner create contract perminssion
	ModifyPerminType_DelGrop
	ModifyPerminType_CrtGrop
	PerminType_AccessContract

)

func init() {
	PerminCache = newImpawnCache()
}

func SetPermConfig(sendTxFlag, crtContractFlag bool) {
	whitelistIsWork_SendTx = sendTxFlag
	whitelistIsWork_CrtContract = crtContractFlag
}

type PerminssionCache struct {
	Cache 		*lru.Cache
	size 		int
}

func newImpawnCache() *PerminssionCache {
	cc := &PerminssionCache{
		size:	20,
	}
	cc.Cache,_ = lru.New(cc.size)
	return cc
}
/////cache

type PerminTable struct {
	LastRootID  int64
	WhiteList  []common.Address
	BlackList  []common.Address
	RootList 	[]common.Address
	PBFT2Root	  map[common.Address]common.Address
	ContractPermi map[common.Address]*ContractListTable  //contract Addr=> Memberlist
	GropPermi	map[common.Address]*GropListTable //group addr => GropListTable
	SendTranPermi map[common.Address]*MemberListTable //Group Addr=> MemberList
	CrtContracetPermi map[common.Address]*MemberListTable //Group Addr => MemberList
	UserBasisPermi  map[common.Address]*BasisPermin   // persion addr => basisperim

}

type MemberListTable struct {
	GroupKey      common.Address
	Id				uint64
	Creator      common.Address
	IsWhitListWork  bool
	WhiteMembers *MemberTable
	BlackMembers *MemberTable
}

type ContractListTable struct {
	GroupKey      common.Address
	Creator      common.Address
	CreateFlag     uint8   //create  is 1, create contractpem is 2, only 2 we can set create flag
	IsWhitListWork  bool
	WhiteMembers *MemberTable
	BlackMembers *MemberTable
}

type GropListTable struct {
	GroupKey      common.Address
	Id				uint64
	Creator      common.Address
	Name 			string
	WhiteMembers *MemberTable
	BlackMembers *MemberTable

}

type MemberTable struct {
	Manager  []*MemberInfo
	Member  []*MemberInfo
}

type  MemberInfo struct {
	MemberID common.Address
	JoinTime       int64
}

type BasisPermin struct {
	MemberID common.Address
	CreatorRoot common.Address
	SendTran    bool
	CrtContract bool
	GropId		 uint64
	GropList    []common.Address
}

func NewPerminTable() *PerminTable {
	return &PerminTable{
		LastRootID:        0,
		WhiteList:         []common.Address{},
		BlackList:         []common.Address{},
		RootList:		    []common.Address{},
		PBFT2Root:		   make(map[common.Address]common.Address),
		ContractPermi:     make(map[common.Address]*ContractListTable),
		GropPermi:         make(map[common.Address]*GropListTable),
		SendTranPermi:     make(map[common.Address]*MemberListTable),
		CrtContracetPermi: make(map[common.Address]*MemberListTable),
		UserBasisPermi:    make(map[common.Address]*BasisPermin),
	}
}

func ClonePerminCaCache(pt *PerminTable) *PerminTable {
	if pt == nil {
		return nil
	}

	tempPT := &PerminTable{
		LastRootID: 		pt.LastRootID,
		WhiteList:         make([]common.Address, len(pt.WhiteList)),
		BlackList:         make([]common.Address, len(pt.BlackList)),
		RootList:			make([]common.Address, len(pt.RootList)),
		PBFT2Root:		   make(map[common.Address]common.Address),
		ContractPermi:     make(map[common.Address]*ContractListTable),
		GropPermi:         make(map[common.Address]*GropListTable),
		SendTranPermi:     make(map[common.Address]*MemberListTable),
		CrtContracetPermi: make(map[common.Address]*MemberListTable),
		UserBasisPermi:    make(map[common.Address]*BasisPermin),
	}
	copy(tempPT.WhiteList, pt.WhiteList)
	copy(tempPT.BlackList, pt.BlackList)
	copy(tempPT.RootList, pt.RootList)

	for k, v := range pt.PBFT2Root {
		tempPT.PBFT2Root[k] = v
	}

	for k, v := range pt.ContractPermi {
		wm := &MemberTable{}
		wmb := &MemberTable{}
		tempPT.ContractPermi[k] = &ContractListTable{v.GroupKey,v.Creator,v.CreateFlag,v.IsWhitListWork,wm,wmb}

		if v.WhiteMembers != nil {
			if len(v.WhiteMembers.Manager)>0{
				for i:=0;i<len(v.WhiteMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.WhiteMembers.Manager[i].MemberID,v.WhiteMembers.Manager[i].JoinTime})
				}
			}
			if len(v.WhiteMembers.Member) >0{
				for i:=0;i<len(v.WhiteMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.WhiteMembers.Member[i].MemberID,v.WhiteMembers.Member[i].JoinTime})
				}
			}
		}

		tempPT.ContractPermi[k].WhiteMembers = wm
		if v.BlackMembers != nil {
			if len(v.BlackMembers.Manager)>0{
				for i:=0;i<len(v.BlackMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.BlackMembers.Manager[i].MemberID,v.BlackMembers.Manager[i].JoinTime})
				}
			}
			if len(v.BlackMembers.Member) >0{
				for i:=0;i<len(v.BlackMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.BlackMembers.Member[i].MemberID,v.BlackMembers.Member[i].JoinTime})
				}
			}
		}
		tempPT.ContractPermi[k].BlackMembers = wmb

	}
	for k, v := range pt.GropPermi {
		wm := &MemberTable{}
		wmb := &MemberTable{}
		tempPT.GropPermi[k] = &GropListTable{v.GroupKey,v.Id,v.Creator,v.Name,wm,wmb}
		if v.WhiteMembers != nil {
			if len(v.WhiteMembers.Manager)>0{
				for i:=0;i<len(v.WhiteMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.WhiteMembers.Manager[i].MemberID,v.WhiteMembers.Manager[i].JoinTime})
				}
			}
			if len(v.WhiteMembers.Member) >0{
				for i:=0;i<len(v.WhiteMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.WhiteMembers.Member[i].MemberID,v.WhiteMembers.Member[i].JoinTime})
				}
			}
		}

		tempPT.GropPermi[k].WhiteMembers = wm
		if v.BlackMembers != nil {
			if len(v.BlackMembers.Manager)>0{
				for i:=0;i<len(v.BlackMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.BlackMembers.Manager[i].MemberID,v.BlackMembers.Manager[i].JoinTime})
				}
			}
			if len(v.BlackMembers.Member) >0{
				for i:=0;i<len(v.BlackMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.BlackMembers.Member[i].MemberID,v.BlackMembers.Member[i].JoinTime})
				}
			}
		}
		tempPT.GropPermi[k].BlackMembers = wmb
	}
	for k, v := range pt.SendTranPermi {
		wm := &MemberTable{}
		wmb := &MemberTable{}
		tempPT.SendTranPermi[k] = &MemberListTable{v.GroupKey,v.Id,v.Creator,v.IsWhitListWork,wm,wmb}
		if v.WhiteMembers != nil {
			if len(v.WhiteMembers.Manager)>0{
				for i:=0;i<len(v.WhiteMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.WhiteMembers.Manager[i].MemberID,v.WhiteMembers.Manager[i].JoinTime})
				}
			}
			if len(v.WhiteMembers.Member) >0{
				for i:=0;i<len(v.WhiteMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.WhiteMembers.Member[i].MemberID,v.WhiteMembers.Member[i].JoinTime})
				}
			}
		}

		tempPT.SendTranPermi[k].WhiteMembers = wm
		if v.BlackMembers != nil {
			if len(v.BlackMembers.Manager)>0{
				for i:=0;i<len(v.BlackMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.BlackMembers.Manager[i].MemberID,v.BlackMembers.Manager[i].JoinTime})
				}
			}
			if len(v.BlackMembers.Member) >0{
				for i:=0;i<len(v.BlackMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.BlackMembers.Member[i].MemberID,v.BlackMembers.Member[i].JoinTime})
				}
			}
		}
		tempPT.SendTranPermi[k].BlackMembers = wmb

	}
	for k, v := range pt.CrtContracetPermi {
		wm := &MemberTable{}
		wmb := &MemberTable{}
		tempPT.CrtContracetPermi[k] = &MemberListTable{v.GroupKey,v.Id,v.Creator,v.IsWhitListWork,wm,wmb}
		if v.WhiteMembers != nil {
			if len(v.WhiteMembers.Manager)>0{
				for i:=0;i<len(v.WhiteMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.WhiteMembers.Manager[i].MemberID,v.WhiteMembers.Manager[i].JoinTime})
				}
			}
			if len(v.WhiteMembers.Member) >0{
				for i:=0;i<len(v.WhiteMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.WhiteMembers.Member[i].MemberID,v.WhiteMembers.Member[i].JoinTime})
				}
			}
		}

		tempPT.CrtContracetPermi[k].WhiteMembers = wm
		if v.BlackMembers != nil {
			if len(v.BlackMembers.Manager)>0{
				for i:=0;i<len(v.BlackMembers.Manager);i++{
					wm.Manager = append(wm.Manager,&MemberInfo{v.BlackMembers.Manager[i].MemberID,v.BlackMembers.Manager[i].JoinTime})
				}
			}
			if len(v.BlackMembers.Member) >0{
				for i:=0;i<len(v.BlackMembers.Member);i++{
					wm.Member = append(wm.Member,&MemberInfo{v.BlackMembers.Member[i].MemberID,v.BlackMembers.Member[i].JoinTime})
				}
			}
		}
		tempPT.CrtContracetPermi[k].BlackMembers = wmb

	}
	for k, v := range pt.UserBasisPermi {

		tempPT.UserBasisPermi[k] = &BasisPermin{v.MemberID,v.CreatorRoot,v.SendTran,v.CrtContract,v.GropId,[]common.Address{}}

		if len(v.GropList)>0{
			tempPT.UserBasisPermi[k].GropList = append(tempPT.UserBasisPermi[k].GropList,v.GropList[:]...)
		}
	}

	return tempPT
}

func (pt *PerminTable) InitPBFTRootGrop(rootAddr []common.Address) {

	var rootImage []common.Address
	lenRoot := len(rootAddr)
	for i:=1;i<=lenRoot;i++{
		rootImgaddr := crypto.CreatePermiRootKey(int64(i))
		rootImage = append(rootImage,rootImgaddr)
		pt.PBFT2Root[rootAddr[i-1]] = rootImgaddr;
	}

	for _, root := range rootAddr {
		//send tx
		key := crypto.CreateGroupkey(root, 1)
		if pt.UserBasisPermi[root] != nil{
			continue
		}
			pt.RootList = append(pt.RootList,root)
			stp := &MemberListTable{Id: 1, GroupKey: key, Creator: root, IsWhitListWork: whitelistIsWork_SendTx, WhiteMembers: &MemberTable{}, BlackMembers: &MemberTable{}}
			pt.SendTranPermi[key] = stp

			//send contract
			key2 := crypto.CreateGroupkey(root, 2)
			stp2 := &MemberListTable{Id: 2, GroupKey: key2, Creator: root, IsWhitListWork: whitelistIsWork_CrtContract, WhiteMembers: &MemberTable{}, BlackMembers: &MemberTable{}}
			pt.CrtContracetPermi[key2] = stp2

			groplist :=[]common.Address{}
			pt.UserBasisPermi[root] = &BasisPermin{MemberID:root,CreatorRoot:root,SendTran:true,CrtContract:true,GropId:0,GropList:groplist}
		}
}

func (pt *PerminTable)UpdataRootInElection(rootAddr, curRootAddr []common.Address) {

	lenRoot := len(rootAddr)
	lenCurRoot := len(curRootAddr)



	for i:=0;i<lenRoot;i++ {
		find := false

		for j := 0; j < lenCurRoot; j++ {
			//find
			if curRootAddr[j] == rootAddr[i] {
				find = true
				break;
			}
		}
		if !find {
			for idex, v := range pt.RootList {
				if v == pt.PBFT2Root[rootAddr[i]] {
					pt.RootList = append(pt.RootList[:idex], pt.RootList[idex:]...)
					delete(pt.PBFT2Root, pt.PBFT2Root[rootAddr[i]])
				}

			}
		}
	}

	for j := 0; j < lenCurRoot; j++ {
		rootImage :=pt.PBFT2Root[curRootAddr[j]]
		if (common.Address{}) == rootImage {
			pt.LastRootID++;
			rootImage = crypto.CreatePermiRootKey(pt.LastRootID)
			pt.RootList = append(pt.RootList,rootImage)
			pt.PBFT2Root[curRootAddr[j]] = rootImage
		}
	}

}

func (pt *PerminTable)GetCreator(from common.Address) common.Address  {

	var crt common.Address

	if pt.UserBasisPermi[from]!=nil{
		crt = pt.UserBasisPermi[from].CreatorRoot
		itotal :=0
		for _,v:= range pt.RootList{
			itotal++
			if crt == pt.PBFT2Root[from] && crt == v{
				return crt
			}
		}
	}

	return crt
}


//Grant Perminission
func (pt *PerminTable)GrantPermission(creator,from,member,gropAddr common.Address, mPermType ModifyPerminType,gropName string ,whitelistisWork bool) (bool ,error)  {


	switch mPermType {
	case ModifyPerminType_AddSendTxPerm:
		return pt.setSendTxPerm(creator,from,member,true)
	case ModifyPerminType_DelSendTxPerm:
		return pt.setSendTxPerm(creator,from,member,false)
	case ModifyPerminType_AddSendTxManagerPerm:
		return pt.setSendTxManagerPerm(creator,from,member,true)
	case ModifyPerminType_DelSendTxManagerPerm:
		return pt.setSendTxManagerPerm(creator,from,member,false)
	case ModifyPerminType_AddCrtContractPerm:
		return pt.setCrtContractPerm(creator,from,member,true)
	case ModifyPerminType_DelCrtContractPerm:
		return pt.setCrtContractPerm(creator,from,member,false)
	case ModifyPerminType_AddCrtContractManagerPerm:
		return pt.setCrtContractManegerPerm(creator,from,member,true)
	case ModifyPerminType_DelCrtContractManagerPerm:
		return pt.setCrtContractManegerPerm(creator,from,member,false)
	case ModifyPerminType_CrtGrop:
		return pt.createGropPerm(from,gropName)
	case ModifyPerminType_DelGrop:
		return pt.delGropPerm(from,gropAddr)
	case ModifyPerminType_AddGropManagerPerm:
		return pt.setGropManagerPerm(gropAddr,member,true)
	case ModifyPerminType_DelGropManagerPerm:
		return pt.setGropManagerPerm(gropAddr,member,false)
	case ModifyPerminType_AddGropMemberPerm:
		 return pt.setGropMemberPerm(gropAddr,member,true)
	case ModifyPerminType_DelGropMemberPerm:
		return pt.setGropMemberPerm(gropAddr,member,false)
	case ModifyPerminType_CrtContractPerm:
		return pt.setContractPem(gropAddr,from,whitelistisWork)
	case ModifyPerminType_AddContractMemberPerm:
		return pt.setContractMember(gropAddr,member,true)
	case ModifyPerminType_DelContractMemberPerm:
		return pt.setContractMember(gropAddr,member,false)
	case ModifyPerminType_AddContractManagerPerm:
		return pt.setContractManager(gropAddr,member,true)
	case ModifyPerminType_DelContractManagerPerm:
		return pt.setContractManager(gropAddr,member,false)
	case ModifyPerminType_AddWhitListPerm:
		pt.WhiteList = append(pt.WhiteList,member)
		break
	case ModifyPerminType_DelWhitListPerm:
		totalM := 0
		for i,w := range pt.WhiteList{
			if w == member{
				pt.WhiteList = append(pt.WhiteList[:i],pt.WhiteList[i+1:]...)
				return true,nil
			}
			totalM++
		}

		if totalM == len(pt.WhiteList){
			return false,MemberNotInGropError
		}
	case ModifyPerminType_AddBlockListPerm:
		pt.BlackList = append(pt.BlackList,member)
		break
	case ModifyPerminType_DelBlockListPerm:
		totalM := 0
		for i,w := range pt.BlackList{
			if w == member{
				pt.BlackList = append(pt.BlackList[:i],pt.BlackList[i+1:]...)
				return true,nil
			}
			totalM++
		}

		if totalM == len(pt.BlackList){
			return false,MemberNotInGropError
		}
		break

	}
	return true,nil
}

func (pt *PerminTable)setSendTxPerm(creator,from ,member common.Address,isAdd bool) (bool,error)  {

	if pt.isInBlackList(from){
		return false,MemberInBlackListError
	}
	//frist time create and sendTx only one grop
	key := crypto.CreateGroupkey(creator,1)
	if pt.SendTranPermi[key] == nil{
		return false,NotRootSendTxError
	}
	if pt.SendTranPermi[key].Id == 0 {
		if from != creator{
			return false,FristGreateGropError
		}
		pt.SendTranPermi[key].Id = 1;
		pt.SendTranPermi[key].GroupKey = key
		pt.SendTranPermi[key].Creator = creator
	}

	//check whitelist frist use whitelistIsWork_SendTx
	iswhitelistWork := pt.SendTranPermi[key].IsWhitListWork
	if iswhitelistWork != whitelistIsWork_SendTx{
		pt.SendTranPermi[key].IsWhitListWork = whitelistIsWork_SendTx
		iswhitelistWork = whitelistIsWork_SendTx
	}

	if isAdd {
		if pt.GropPermi[member] == nil{

			if pt.UserBasisPermi[member] == nil{
				pt.UserBasisPermi[member] = &BasisPermin{}
				pt.UserBasisPermi[member].MemberID = member
				pt.UserBasisPermi[member].CreatorRoot = creator
			}

			pt.UserBasisPermi[member].SendTran = true


		}else{
			if !pt.SetGropMemberRoot(member,creator){
				return false,MemberAreadInGropError
			}

		}



		if iswhitelistWork{

			if pt.SendTranPermi[key].WhiteMembers == nil || pt.SendTranPermi[key].WhiteMembers.Member != nil{



				for _,m := range pt.SendTranPermi[key].WhiteMembers.Member{
					if m.MemberID == member{
						return false,MemberAreadInGropError
					}
				}
			}

			mber := &MemberInfo{member,0}
			pt.SendTranPermi[key].WhiteMembers.Member = append(pt.SendTranPermi[key].WhiteMembers.Member,mber )




		}
	}else{
		if !pt.UserBasisPermi[member].SendTran || pt.UserBasisPermi[member].MemberID != member {
			//data base is nill
			//return false,MemberNotInGropError
		}

		pt.UserBasisPermi[member].SendTran = false

		if iswhitelistWork{

			totalM :=0;
			if pt.SendTranPermi[key].WhiteMembers.Member != nil{
			for i,m := range pt.SendTranPermi[key].WhiteMembers.Member{
				if m.MemberID == member{
					pt.SendTranPermi[key].WhiteMembers.Member = append(pt.SendTranPermi[key].WhiteMembers.Member[:i],pt.SendTranPermi[key].WhiteMembers.Member[i+1:]...)
					return  true,nil
				}
				totalM++
			}
			}

			if totalM == len(pt.SendTranPermi[key].WhiteMembers.Member){
				return false,MemberNotInGropError
			}
		}else{

			if pt.SendTranPermi[key].BlackMembers.Member != nil{


			for _,m := range pt.SendTranPermi[key].BlackMembers.Member{
				if m.MemberID == member{
					return false,MemberAreadInGropError
				}

			}
			}
			mber := &MemberInfo{member,0}
			pt.SendTranPermi[key].BlackMembers.Member = append(pt.SendTranPermi[key].BlackMembers.Member,mber )

		}
	}

	return true, nil
}

func (pt *PerminTable)isInBlackList(from common.Address)(bool)  {
	//check black list
	for _,b := range pt.BlackList {
		if b == from{ return true}
	}
	return false
}

func (pt *PerminTable)setSendTxManagerPerm(creator,from ,member common.Address,isAdd bool) (bool,error)  {

	if pt.isInBlackList(from){
		return false,MemberInBlackListError
	}
	//frist time create and sendTx only one grop
	key := crypto.CreateGroupkey(creator,1)
	if pt.SendTranPermi[key].Id == 0 {
		if from != creator{
			return false,FristGreateGropError
		}
		pt.SendTranPermi[key].Id = 1;
		pt.SendTranPermi[key].GroupKey = key
		pt.SendTranPermi[key].Creator = creator
	}

	//check whitelist frist use whitelistIsWork_SendTx
	iswhitelistWork := pt.SendTranPermi[key].IsWhitListWork
	if iswhitelistWork != whitelistIsWork_SendTx{
		pt.SendTranPermi[key].IsWhitListWork = whitelistIsWork_SendTx
	}

	if pt.UserBasisPermi[member] == nil{
		pt.UserBasisPermi[member] = &BasisPermin{}
		pt.UserBasisPermi[member].CreatorRoot = creator
	}

	if isAdd {
		if !pt.UserBasisPermi[member].SendTran || pt.UserBasisPermi[member].MemberID != member {
			//data base is nill
			//return false,MemberNotSendTxPermError
			/*if pt.UserBasisPermi[member].MemberID != member{
				pt.UserBasisPermi[member].MemberID = member
			}
			pt.UserBasisPermi[member].SendTran = true;*/
		}

		if iswhitelistWork{
			if pt.SendTranPermi[key].WhiteMembers.Manager != nil{

				for _,m := range pt.SendTranPermi[key].WhiteMembers.Manager{
					if m.MemberID == member{
						return false,MemberAreadInGropError
					}
				}
			}

			mber := &MemberInfo{member,0}
			pt.SendTranPermi[key].WhiteMembers.Manager = append(pt.SendTranPermi[key].WhiteMembers.Manager,mber )
		}else{
			if pt.SendTranPermi[key].BlackMembers.Manager != nil{

				for _,m := range pt.SendTranPermi[key].BlackMembers.Manager{
					if m.MemberID == member{
						return false,MemberAreadInGropError
					}
				}
			}

			mber := &MemberInfo{member,0}
			pt.SendTranPermi[key].BlackMembers.Manager = append(pt.SendTranPermi[key].BlackMembers.Manager,mber )

		}


	}else{
		if !pt.UserBasisPermi[member].SendTran || pt.UserBasisPermi[member].MemberID != member {
			//data base is nill
			//return false,MemberNotInGropError
		}

		if iswhitelistWork{

			totalM :=0;
			for i,m := range pt.SendTranPermi[key].WhiteMembers.Manager{
				if m.MemberID == member{
					pt.SendTranPermi[key].WhiteMembers.Manager = append(pt.SendTranPermi[key].WhiteMembers.Manager[:i],pt.SendTranPermi[key].WhiteMembers.Manager[i+1:]...)
					return  true,nil
				}
				totalM++
			}

			if totalM == len(pt.SendTranPermi[key].WhiteMembers.Manager){
				return false,MemberNotInGropError
			}


		}else{
			totalM :=0;
			for i,m := range pt.SendTranPermi[key].BlackMembers.Manager{
				if m.MemberID == member{
					pt.SendTranPermi[key].BlackMembers.Manager = append(pt.SendTranPermi[key].BlackMembers.Manager[:i],pt.SendTranPermi[key].BlackMembers.Manager[i+1:]...)
					return  true,nil
				}
				totalM++
			}

			if totalM == len(pt.SendTranPermi[key].BlackMembers.Manager){
				return false,MemberNotInGropError
			}
		}

	}

	return true, nil
}

func (pt *PerminTable)SetGropMemberRoot(gropAddr,creator common.Address) bool{
	if pt.GropPermi[gropAddr] == nil{
		return false
	}

	for _,v := range  pt.GropPermi[gropAddr].WhiteMembers.Manager{
		if pt.GropPermi[v.MemberID] != nil{
			pt.SetGropMemberRoot(v.MemberID,creator)
		}else{
			pt.UserBasisPermi[v.MemberID].CreatorRoot = creator
		}
	}

	for _,v := range  pt.GropPermi[gropAddr].WhiteMembers.Member{
		if pt.GropPermi[v.MemberID] !=nil{
			pt.SetGropMemberRoot(v.MemberID,creator)
		}else{
			pt.UserBasisPermi[v.MemberID].CreatorRoot = creator
		}
	}

	return true

}

func (pt *PerminTable)setCrtContractPerm(creator,from ,member common.Address,isAdd bool) (bool,error){

	if pt.isInBlackList(from){
		return false,MemberInBlackListError
	}

	/*if pt.ContractPermi[contractAddr] == nil{
		return false,ContractNotCreatePremError
	}
	creator := pt.ContractPermi[contractAddr].Creator*/

	//frist time create and sendTx only one grop
	key := crypto.CreateGroupkey(creator,2)
	if pt.CrtContracetPermi[key] == nil{
		if from != creator{
			return false,FristGreateGropError
		}
		pt.CrtContracetPermi[key] = &MemberListTable{WhiteMembers: &MemberTable{}, BlackMembers: &MemberTable{}}
		pt.CrtContracetPermi[key].Id = 2;
		pt.CrtContracetPermi[key].GroupKey = key
		pt.CrtContracetPermi[key].Creator = creator
		pt.CrtContracetPermi[key].WhiteMembers = &MemberTable{}
	}

	//check whitelist frist use whitelistIsWork_SendTx
	iswhitelistWork := pt.CrtContracetPermi[key].IsWhitListWork
	if iswhitelistWork != whitelistIsWork_CrtContract{
		pt.CrtContracetPermi[key].IsWhitListWork = whitelistIsWork_CrtContract
		iswhitelistWork = whitelistIsWork_CrtContract
	}

	if isAdd{
		if pt.UserBasisPermi[member] == nil {
			pt.UserBasisPermi[member] = &BasisPermin{}
			pt.UserBasisPermi[member].MemberID = member
			pt.UserBasisPermi[member].CreatorRoot = creator

		}
		if !pt.UserBasisPermi[member].CrtContract || pt.UserBasisPermi[member].MemberID != member {
			//data base is nill
			//return false,MemberNotInGropError
		}
		pt.UserBasisPermi[member].CrtContract = true

		if iswhitelistWork{
			if pt.CrtContracetPermi[key].WhiteMembers.Member != nil{

				for _,m := range pt.CrtContracetPermi[key].WhiteMembers.Member{
					if m.MemberID == member{
						return false,MemberAreadInGropError
					}
				}
			}

			mber := &MemberInfo{member,0}
			pt.CrtContracetPermi[key].WhiteMembers.Member = append(pt.CrtContracetPermi[key].WhiteMembers.Member,mber )

		}

	}else{
		if !pt.UserBasisPermi[member].CrtContract || pt.UserBasisPermi[member].MemberID != member {
			//data base is nill
			return false,MemberNotInGropError
		}
		pt.UserBasisPermi[member].CrtContract = false

		if iswhitelistWork{

			totalM :=0;
			if pt.CrtContracetPermi[key].WhiteMembers.Member != nil {
				for i, m := range pt.CrtContracetPermi[key].WhiteMembers.Member {
					if m.MemberID == member {
						pt.CrtContracetPermi[key].WhiteMembers.Member = append(pt.CrtContracetPermi[key].WhiteMembers.Member[:i], pt.CrtContracetPermi[key].WhiteMembers.Member[i+1:]...)
						return true, nil
					}
					totalM++
				}
			}

			if totalM == len(pt.CrtContracetPermi[key].WhiteMembers.Member){
				return false,MemberNotInGropError
			}
		}else{

			if pt.CrtContracetPermi[key].BlackMembers.Member != nil {
			for _,m := range pt.CrtContracetPermi[key].BlackMembers.Member{
				if m.MemberID == member{
					return false,MemberAreadInGropError
				}

			}}
			mber := &MemberInfo{member,0}
			pt.CrtContracetPermi[key].BlackMembers.Member = append(pt.CrtContracetPermi[key].BlackMembers.Member,mber )

		}

	}

	return true,nil

}

func (pt *PerminTable)setCrtContractManegerPerm(creator,from ,member common.Address,isAdd bool) (bool,error){

	if pt.isInBlackList(from){
		return false,MemberInBlackListError
	}


	//creator := pt.ContractPermi[contractAddr].Creator

	//frist time create and sendTx only one grop
	key := crypto.CreateGroupkey(creator,2)
	if pt.CrtContracetPermi[key] == nil{
		pt.CrtContracetPermi[key] = &MemberListTable{WhiteMembers: &MemberTable{}, BlackMembers: &MemberTable{}}
	}
	if pt.CrtContracetPermi[key].Id == 0 {
		if from != creator{
			return false,FristGreateGropError
		}
		pt.CrtContracetPermi[key].Id = 2;
		pt.CrtContracetPermi[key].GroupKey = key
		pt.CrtContracetPermi[key].Creator = creator
	}

	//check whitelist frist use whitelistIsWork_SendTx
	iswhitelistWork := pt.CrtContracetPermi[key].IsWhitListWork
	if iswhitelistWork != whitelistIsWork_SendTx{
		pt.CrtContracetPermi[key].IsWhitListWork = whitelistIsWork_SendTx
		iswhitelistWork = whitelistIsWork_SendTx
	}

	//check member owner create contract
	if !pt.UserBasisPermi[member].CrtContract || pt.UserBasisPermi[member].MemberID != member {
		//data base is nill
		//return false,MemberNotInGropError
	}

	if isAdd{
		if pt.UserBasisPermi[member] == nil {
			pt.UserBasisPermi[member] = &BasisPermin{}
			pt.UserBasisPermi[member].MemberID = member
			pt.UserBasisPermi[member].CreatorRoot = creator

		}
		if iswhitelistWork{


			if pt.CrtContracetPermi[key].WhiteMembers.Manager != nil{

			for _,m := range pt.CrtContracetPermi[key].WhiteMembers.Manager{
				if m.MemberID == member{
					return false,MemberAreadInGropError
				}
			}

			}

			mber := &MemberInfo{member,0}
			pt.CrtContracetPermi[key].WhiteMembers.Manager = append(pt.CrtContracetPermi[key].WhiteMembers.Manager,mber )

			//pt.UserBasisPermi[member].CrtContract = true
		}else{
			if pt.CrtContracetPermi[key].BlackMembers.Manager != nil{

				for _,m := range pt.CrtContracetPermi[key].BlackMembers.Manager{
					if m.MemberID == member{
						return false,MemberAreadInGropError
					}
				}

			}

			mber := &MemberInfo{member,0}
			pt.CrtContracetPermi[key].BlackMembers.Manager = append(pt.CrtContracetPermi[key].BlackMembers.Manager,mber )
		}
	}else{


		if iswhitelistWork {
			totalM := 0;
			if pt.CrtContracetPermi[key].WhiteMembers.Manager != nil {
				for i, m := range pt.CrtContracetPermi[key].WhiteMembers.Manager {
					if m.MemberID == member {
						pt.CrtContracetPermi[key].WhiteMembers.Manager = append(pt.CrtContracetPermi[key].WhiteMembers.Manager[:i], pt.CrtContracetPermi[key].WhiteMembers.Manager[i+1:]...)
						return true, nil
					}
					totalM++
				}
			}

			if totalM == len(pt.CrtContracetPermi[key].WhiteMembers.Manager) {
				return false, MemberNotInGropError
			}
		}else{
			totalM := 0;
			if pt.CrtContracetPermi[key].BlackMembers.Manager != nil {
				for i, m := range pt.CrtContracetPermi[key].BlackMembers.Manager {
					if m.MemberID == member {
						pt.CrtContracetPermi[key].BlackMembers.Manager = append(pt.CrtContracetPermi[key].BlackMembers.Manager[:i], pt.CrtContracetPermi[key].BlackMembers.Manager[i+1:]...)
						return true, nil
					}
					totalM++
				}
			}

			if totalM == len(pt.CrtContracetPermi[key].BlackMembers.Manager) {
				return false, MemberNotInGropError
			}
		}

	}

	return true,nil

}

func (pt *PerminTable) createGropPerm(creator common.Address, gropName string) (bool,error) {



	if len(gropName) == 0{
		return false, errors.New("Grop name len is zero")
	}
	if pt.UserBasisPermi[creator] == nil{
		return false,GropNotExitError
	}
	id := pt.UserBasisPermi[creator].GropId
	if id == 0{
		pt.UserBasisPermi[creator].GropId = 3
		id = 3
	}else{
		pt.UserBasisPermi[creator].GropId++
		id++
	}
	key := crypto.CreateGroupkey(creator,id)

	if pt.GropPermi[key] == nil{
		pt.GropPermi[key] = &GropListTable{WhiteMembers: &MemberTable{}, BlackMembers: &MemberTable{}}
	}

	if pt.UserBasisPermi[creator].GropList != nil{


		for _,gropAddr := range pt.UserBasisPermi[creator].GropList{
			if gropName == pt.GropPermi[gropAddr].Name{
				return false,GropNameAlreadyUseError
			}
		}

	}

	pt.GropPermi[key].Name = gropName
	pt.GropPermi[key].Creator = creator
	pt.GropPermi[key].GroupKey = key
	pt.GropPermi[key].Id = id

	pt.UserBasisPermi[creator].GropList = append(pt.UserBasisPermi[creator].GropList,key)

	return true,nil

}

func (pt *PerminTable)GetLastGroupAddr(from common.Address) common.Address  {
	if pt.UserBasisPermi[from] == nil{
		return common.Address{}
	}

	id := pt.UserBasisPermi[from].GropId
	return crypto.CreateGroupkey(from,id)
}

func (pt *PerminTable) delGropPerm(from,gropAddr common.Address) (bool,error){

	/*if pt.UserBasisPermi[from] == nil{
		return false,GropNotExitError
	}*/

	if pt.GropPermi[gropAddr] == nil{
		return false,GropNotExitError
	}
	if pt.UserBasisPermi[from] != nil{

	for i,g := range pt.UserBasisPermi[from].GropList{
		if g == gropAddr{
			pt.UserBasisPermi[from].GropList = append(pt.UserBasisPermi[from].GropList[:i],pt.UserBasisPermi[from].GropList[i+1:]...)
			pt.delGropMember(gropAddr)
			delete(pt.GropPermi,gropAddr)
			return true,nil
		}

	}

	}

		pt.delGropMember(gropAddr)
		delete(pt.GropPermi,gropAddr)
		return true,nil



}

func (pt *PerminTable) delGropMember(gropAddr common.Address) bool  {
	if pt.GropPermi[gropAddr].WhiteMembers != nil{
		if pt.GropPermi[gropAddr].WhiteMembers.Manager != nil{
			for _,k:= range pt.GropPermi[gropAddr].WhiteMembers.Manager{
				if pt.GropPermi[k.MemberID] != nil{
					continue
				}
				for i:=0;i<len(pt.UserBasisPermi[k.MemberID].GropList);i++{
					if pt.UserBasisPermi[k.MemberID].GropList[i] == gropAddr{
						pt.UserBasisPermi[k.MemberID].GropList = append(pt.UserBasisPermi[k.MemberID].GropList[:i],pt.UserBasisPermi[k.MemberID].GropList[i+1:]...)
					}
				}

			}
		}
		if pt.GropPermi[gropAddr].WhiteMembers.Member != nil{
			for _,k:= range pt.GropPermi[gropAddr].WhiteMembers.Member{
				if pt.GropPermi[k.MemberID] != nil{
					continue
				}
				for i:=0;i<len(pt.UserBasisPermi[k.MemberID].GropList);i++{
					if pt.UserBasisPermi[k.MemberID].GropList[i] == gropAddr{
						pt.UserBasisPermi[k.MemberID].GropList = append(pt.UserBasisPermi[k.MemberID].GropList[:i],pt.UserBasisPermi[k.MemberID].GropList[i+1:]...)
					}
				}

			}
		}
	}
	if pt.GropPermi[gropAddr].BlackMembers != nil{
		if pt.GropPermi[gropAddr].BlackMembers.Manager != nil{
			for _,k:= range pt.GropPermi[gropAddr].BlackMembers.Manager{
				if pt.GropPermi[k.MemberID] != nil{
					continue
				}
				for i:=0;i<len(pt.UserBasisPermi[k.MemberID].GropList);i++{
					if pt.UserBasisPermi[k.MemberID].GropList[i] == gropAddr{
						pt.UserBasisPermi[k.MemberID].GropList = append(pt.UserBasisPermi[k.MemberID].GropList[:i],pt.UserBasisPermi[k.MemberID].GropList[i+1:]...)
					}
				}

			}
		}
		if pt.GropPermi[gropAddr].BlackMembers.Member != nil{
			for _,k:= range pt.GropPermi[gropAddr].BlackMembers.Member{
				if pt.GropPermi[k.MemberID] != nil{
					continue
				}
				for i:=0;i<len(pt.UserBasisPermi[k.MemberID].GropList);i++{
					if pt.UserBasisPermi[k.MemberID].GropList[i] == gropAddr{
						pt.UserBasisPermi[k.MemberID].GropList = append(pt.UserBasisPermi[k.MemberID].GropList[:i],pt.UserBasisPermi[k.MemberID].GropList[i+1:]...)
					}
				}

			}
		}
	}
	return true
}

func (pt *PerminTable) setGropMemberPerm(gropAddr ,member common.Address,isAdd bool) (bool,error){

	if pt.isInBlackList(member){
		return false,MemberInBlackListError
	}

	if pt.GropPermi[gropAddr] == nil{
		return false,GropNotExitError
	}
	if pt.GropPermi[gropAddr].GroupKey != gropAddr{
		return false,GropNotExitError
	}

	if isAdd{

		/*if pt.UserBasisPermi[member] == nil{
			return false,MemberNotSentTXPerm
		}*/

		for _,m :=range  pt.GropPermi[gropAddr].WhiteMembers.Member{
			if m.MemberID == member{
				return false, MemberAreadInGropError
			}
		}

		mst := &MemberInfo{member,0}
		pt.GropPermi[gropAddr].WhiteMembers.Member = append(pt.GropPermi[gropAddr].WhiteMembers.Member,mst)

		if pt.UserBasisPermi[member] == nil{
			pt.UserBasisPermi[member] = &BasisPermin{member,common.Address{},false,false,0,[]common.Address{gropAddr}}
		}

	}else{

		totalM :=0;
		for i,m :=range  pt.GropPermi[gropAddr].WhiteMembers.Member{
			if m.MemberID == member{
				pt.GropPermi[gropAddr].WhiteMembers.Member = append(pt.GropPermi[gropAddr].WhiteMembers.Member[:i],pt.GropPermi[gropAddr].WhiteMembers.Member[i+1:]...)
				return true,nil
			}
			totalM++
		}

		if totalM == len(pt.GropPermi[gropAddr].WhiteMembers.Member){
			return false,MemberNotInGropError
		}

	}

	return true,nil
}

func (pt *PerminTable) setGropManagerPerm(gropAddr ,manager common.Address,isAdd bool) (bool,error){

	if pt.isInBlackList(manager){
		return false,MemberInBlackListError
	}
	if pt.GropPermi[gropAddr] == nil{
		return false,GropNotExitError
	}


	if pt.GropPermi[gropAddr].GroupKey != gropAddr{
		return false,GropNotExitError
	}

	if  pt.GropPermi[gropAddr].WhiteMembers == nil{
		pt.GropPermi[gropAddr].WhiteMembers = &MemberTable{}
	}
	if isAdd{
		if pt.GropPermi[gropAddr].WhiteMembers.Manager != nil{

		for _,m :=range  pt.GropPermi[gropAddr].WhiteMembers.Manager{
			if m.MemberID == manager{
				return false, MemberAreadInGropError
			}
		}
		}

		mst := &MemberInfo{manager,0}
		pt.GropPermi[gropAddr].WhiteMembers.Manager = append(pt.GropPermi[gropAddr].WhiteMembers.Manager,mst)

		if pt.UserBasisPermi[manager] == nil{
			pt.UserBasisPermi[manager] = &BasisPermin{manager,common.Address{},false,false,0,[]common.Address{gropAddr}}
		}

	}else{

		totalM :=0;
		for i,m :=range  pt.GropPermi[gropAddr].WhiteMembers.Manager{
			if m.MemberID == manager{
				pt.GropPermi[gropAddr].WhiteMembers.Manager = append(pt.GropPermi[gropAddr].WhiteMembers.Manager[:i],pt.GropPermi[gropAddr].WhiteMembers.Manager[i+1:]...)
				return true,nil
			}
			totalM++
		}

		if totalM == len(pt.GropPermi[gropAddr].WhiteMembers.Manager){
			return false,MemberNotInGropError
		}

	}
	return true,nil
}

func CreateContractPemimission(contractAddr ,creator common.Address,nonce uint64,statedb StateDB )(bool,error) {
	permTable := NewPerminTable()
	err := permTable.Load(statedb)
	if err != nil {
		return false,errors.New("load permiTable fail")
	}
	if _,err :=permTable.CreateContractPem(contractAddr,creator,nonce,false); err != nil{
		return false ,err
	}

	permTable.Save(statedb)

	return  true,nil
}

func (pt *PerminTable) CreateContractPem(contractAddr ,creator common.Address,nonce uint64 ,isAdd bool) (bool,error) {
	if contractAddr != crypto.CreateAddress(creator,nonce){
		return false, errors.New("CreateContractPem fail gropAddr not equl contract Addr")
	}

	pt.ContractPermi[contractAddr] = &ContractListTable{contractAddr,creator,1,whitelistIsWork_SendTx,&MemberTable{},&MemberTable{}}

	return true,nil
}

func (pt *PerminTable) setContractPem(contractAddr ,creator common.Address, whitelistisWork bool) (bool,error)  {


	if pt.ContractPermi[contractAddr] == nil{
		return false,ContractNotCreatePremError
	}

	if  pt.ContractPermi[contractAddr].CreateFlag != 1 || pt.ContractPermi[contractAddr].Creator != creator{
		return false , ContractAlreadyCreatePremError
	}

	pt.ContractPermi[contractAddr].CreateFlag = uint8(2)
	pt.ContractPermi[contractAddr].IsWhitListWork = whitelistisWork

	return true,nil
}

func (pt *PerminTable) setContractMember(contractAddr,member common.Address,isAdd bool)(bool,error){

	if pt.isInBlackList(member){
		return false,MemberInBlackListError
	}
	if pt.ContractPermi[contractAddr] == nil{
		return false,ContractNotCreatePremError
	}

	if pt.ContractPermi[contractAddr].CreateFlag != 2{
		return false,ContractPremFlagError
	}
	if isAdd{
		if pt.ContractPermi[contractAddr].IsWhitListWork{

			for _,m := range pt.ContractPermi[contractAddr].WhiteMembers.Member{
				if m.MemberID == member{
					return false,MemberAreadInGropError
				}
			}

			mem :=&MemberInfo{member,0}
			pt.ContractPermi[contractAddr].WhiteMembers.Member = append(pt.ContractPermi[contractAddr].WhiteMembers.Member,mem)
		}
	}else{
		if pt.ContractPermi[contractAddr].IsWhitListWork{

			totalM :=0;
			for i,m := range pt.ContractPermi[contractAddr].WhiteMembers.Member{
				if m.MemberID == member{
					pt.ContractPermi[contractAddr].WhiteMembers.Member = append(pt.ContractPermi[contractAddr].WhiteMembers.Member[:i],pt.ContractPermi[contractAddr].WhiteMembers.Member[i+1:]...)
					return true,nil
				}
				totalM ++;
			}

			if totalM == len(pt.ContractPermi[contractAddr].WhiteMembers.Member){
				return false,MemberNotInGropError
			}

		}else{
			for _,m := range pt.ContractPermi[contractAddr].BlackMembers.Member{
				if m.MemberID == member{
					return false,MemberAreadInGropError
				}
			}

			mem :=&MemberInfo{member,0}
			pt.ContractPermi[contractAddr].BlackMembers.Member = append(pt.ContractPermi[contractAddr].BlackMembers.Member,mem)
		}
	}

	return true,nil
}

func (pt *PerminTable) setContractManager(contractAddr,manager common.Address,isAdd bool)(bool,error){

	if pt.isInBlackList(manager){
		return false,MemberInBlackListError
	}

	if pt.ContractPermi[contractAddr] == nil{
		return false,ContractNotCreatePremError
	}

	if pt.ContractPermi[contractAddr].CreateFlag != 2{
		return false,ContractPremFlagError
	}
	if isAdd{
		if pt.ContractPermi[contractAddr].IsWhitListWork{
			if pt.ContractPermi[contractAddr].WhiteMembers.Manager != nil{


			for _,m := range pt.ContractPermi[contractAddr].WhiteMembers.Manager{
				if m.MemberID == manager{
					return false,MemberAreadInGropError
				}
			}
			}

			mem :=&MemberInfo{manager,0}
			pt.ContractPermi[contractAddr].WhiteMembers.Manager = append(pt.ContractPermi[contractAddr].WhiteMembers.Manager,mem)
		}else{
			if pt.ContractPermi[contractAddr].BlackMembers.Manager != nil{


				for _,m := range pt.ContractPermi[contractAddr].BlackMembers.Manager{
					if m.MemberID == manager{
						return false,MemberAreadInGropError
					}
				}
			}

			mem :=&MemberInfo{manager,0}
			pt.ContractPermi[contractAddr].BlackMembers.Manager = append(pt.ContractPermi[contractAddr].BlackMembers.Manager,mem)

		}
	}else{
		if pt.ContractPermi[contractAddr].IsWhitListWork{

			totalM :=0;
			if pt.ContractPermi[contractAddr].WhiteMembers.Manager != nil{


			for i,m := range pt.ContractPermi[contractAddr].WhiteMembers.Manager{
				if m.MemberID == manager{
					pt.ContractPermi[contractAddr].WhiteMembers.Manager = append(pt.ContractPermi[contractAddr].WhiteMembers.Manager[:i],pt.ContractPermi[contractAddr].WhiteMembers.Manager[i+1:]...)
					return true,nil
				}
				totalM ++;
			}
			}

			if totalM == len(pt.ContractPermi[contractAddr].WhiteMembers.Manager){
				return false,MemberNotInGropError
			}

		}else{
			totalM :=0;
			if pt.ContractPermi[contractAddr].BlackMembers.Manager != nil{


				for i,m := range pt.ContractPermi[contractAddr].BlackMembers.Manager{
					if m.MemberID == manager{
						pt.ContractPermi[contractAddr].BlackMembers.Manager = append(pt.ContractPermi[contractAddr].BlackMembers.Manager[:i],pt.ContractPermi[contractAddr].BlackMembers.Manager[i+1:]...)
						return true,nil
					}
					totalM ++;
				}
			}

			if totalM == len(pt.ContractPermi[contractAddr].BlackMembers.Manager){
				return false,MemberNotInGropError
			}
		}
	}
	return true,nil
}


func (pt *PerminTable)CheckActionPerm(from,gropAddr,contractAddr common.Address, mPermType ModifyPerminType) bool{

	//check black list
	for _,b := range pt.BlackList {
		if b == from{ return false}
	}

	//check whitlist
	for _,b := range pt.WhiteList {
		if b == from{ return true}
	}

	creator := pt.GetCreator(from)
	if creator == (common.Address{}) {
		return false
	}

	switch mPermType {
	case ModifyPerminType_AddSendTxPerm ,
	ModifyPerminType_DelSendTxPerm ,
	ModifyPerminType_AddSendTxManagerPerm ,
	ModifyPerminType_DelSendTxManagerPerm:
		return pt.checkSendTxManager(from,creator)
	case ModifyPerminType_AddCrtContractPerm,
	ModifyPerminType_DelCrtContractPerm,
	ModifyPerminType_AddCrtContractManagerPerm,
	ModifyPerminType_DelCrtContractManagerPerm:
		return pt.checkCrtContractManager(from,creator)
	case ModifyPerminType_AddGropManagerPerm,
	ModifyPerminType_DelGropManagerPerm,
	ModifyPerminType_AddGropMemberPerm,
	ModifyPerminType_DelGropMemberPerm:
		if pt.GropPermi[gropAddr] == nil{
			return false
		}
		if from == pt.GropPermi[gropAddr].Creator{return true}
		for _,g := range pt.GropPermi[gropAddr].WhiteMembers.Manager{
			if g.MemberID == from{return true}
		}
		break
	case ModifyPerminType_CrtContractPerm:
		if pt.ContractPermi[contractAddr] == nil{
			return false
		}
		if from == pt.ContractPermi[contractAddr].Creator{return true}
		break
	case ModifyPerminType_AddContractMemberPerm,
	ModifyPerminType_DelContractMemberPerm,
	ModifyPerminType_AddContractManagerPerm,
	ModifyPerminType_DelContractManagerPerm:
		if pt.ContractPermi[contractAddr] == nil{
			return false
		}
		if from == pt.ContractPermi[contractAddr].Creator{return true}
		if !pt.ContractPermi[contractAddr].IsWhitListWork{
			for _,c := range pt.ContractPermi[contractAddr].BlackMembers.Manager{
				if c.MemberID == from{return false}
			}
			return true
		}else{
			for _,c := range pt.ContractPermi[contractAddr].WhiteMembers.Manager{
				if c.MemberID == from{return true}
			}
		}
		break
	case PerminType_SendTx,ModifyPerminType_CrtGrop:

		v,ok := pt.UserBasisPermi[from]
		if ok && v.SendTran {
			return true
		}else{
			return pt.checkSendTx(from,creator)
		}
		break
	case ModifyPerminType_DelGrop:

		if pt.GropPermi[gropAddr] == nil{
			return false
		}
		/*if pt.UserBasisPermi[from] != nil {

			if pt.UserBasisPermi[from].GropList != nil {
				for _, g := range pt.UserBasisPermi[from].GropList {
					if g == gropAddr {
						return true
					}
				}
			}
		}*/


			if pt.GropPermi[gropAddr].Creator == from{
				return true
			}

			if pt.GropPermi[gropAddr].WhiteMembers.Manager != nil{
				for _,g := range pt.GropPermi[gropAddr].WhiteMembers.Manager {
					if g.MemberID == from{
						return true
					}
				}
			}
			//}


		break
	case PerminType_CreateContract:

		if pt.UserBasisPermi[from] == nil{
			return false
		}

		if pt.UserBasisPermi[from].CrtContract  {
			return true
		}else{
			return pt.checkCrtContract(from,creator)
		}

	case PerminType_AccessContract:

		if pt.ContractPermi[contractAddr] == nil{
			return false
		}

		if pt.UserBasisPermi[from] == nil{
			return false
		}

		if pt.UserBasisPermi[from].SendTran != true {
			return false
		}else{
			if !pt.checkSendTx(from,creator){
				return false
			}
		}

		 if pt.ContractPermi[contractAddr].Creator == from{
		 	return true
		 }

		if !pt.ContractPermi[contractAddr].IsWhitListWork{
			for _ ,c := range pt.ContractPermi[contractAddr].BlackMembers.Member{
				if c.MemberID == from{return false}
			}
			for _ ,c := range pt.ContractPermi[contractAddr].BlackMembers.Manager{
				if c.MemberID == from{return true}
			}
		}else{
			for _ ,c := range pt.ContractPermi[contractAddr].WhiteMembers.Manager{
				if c.MemberID == from{return true}
			}
			for _ ,c := range pt.ContractPermi[contractAddr].WhiteMembers.Member{
				if c.MemberID == from{return true}
			}
		}

	}
	return false
}

func (pt *PerminTable)checkCrtContractManager(from,creator common.Address) bool{
	key := crypto.CreateGroupkey(creator,2)

	if pt.CrtContracetPermi[key] == nil{
		return false
	}
	if from == pt.CrtContracetPermi[key].Creator  {
		return true
	}

	if !pt.CrtContracetPermi[key].IsWhitListWork{
		if pt.CrtContracetPermi[key].BlackMembers.Manager == nil{
			return false
		}
		for _ ,m := range pt.CrtContracetPermi[key].BlackMembers.Manager{
			//need check MemberID is grop id or not
			if pt.GropPermi[m.MemberID] != nil{
				return pt.findMember(m.MemberID,from,false)
			}else{
				if m.MemberID == from{return true}
			}

		}
	}else{
		if pt.CrtContracetPermi[key].WhiteMembers.Manager == nil{
			return false
		}
		for _ ,m := range pt.CrtContracetPermi[key].WhiteMembers.Manager{
			if m.MemberID == from{return true}
		}
	}
	return false
}
func (pt *PerminTable)checkCrtContract(from,creator common.Address) bool{
	key := crypto.CreateGroupkey(creator,2)

	if pt.CrtContracetPermi[key] == nil{
		return false
	}
	if from == pt.CrtContracetPermi[key].Creator  {
		return true
	}

	if !pt.CrtContracetPermi[key].IsWhitListWork{
		if pt.CrtContracetPermi[key].BlackMembers.Manager == nil{

		for _ ,m := range pt.CrtContracetPermi[key].BlackMembers.Manager{
			//need check MemberID is grop id or not
			if pt.GropPermi[m.MemberID] != nil{
				return pt.findMember(m.MemberID,from,false)
			}else{
				if m.MemberID == from{return false}
			}

		}

		}
		if pt.CrtContracetPermi[key].BlackMembers.Member != nil{

		for _ ,m := range pt.CrtContracetPermi[key].BlackMembers.Member{
			//need check MemberID is grop id or not
			if pt.GropPermi[m.MemberID] != nil{
				return pt.findMember(m.MemberID,from,false)
			}else{
				if m.MemberID == from{return false}
			}

		}

		}


	}else{
		if pt.CrtContracetPermi[key].WhiteMembers.Manager != nil{

			for _ ,m := range pt.CrtContracetPermi[key].WhiteMembers.Manager{
				//need check MemberID is grop id or not
				if pt.GropPermi[m.MemberID] != nil{
					return pt.findMember(m.MemberID,from,false)
				}else{
					if m.MemberID == from{return true}
				}

			}

		}
		if pt.CrtContracetPermi[key].WhiteMembers.Member != nil{

			for _ ,m := range pt.CrtContracetPermi[key].WhiteMembers.Member{
				//need check MemberID is grop id or not
				if pt.GropPermi[m.MemberID] != nil{
					return pt.findMember(m.MemberID,from,false)
				}else{
					if m.MemberID == from{return true}
				}

			}

		}
	}
	return false
}

func (pt *PerminTable)checkSendTxManager(from,creator common.Address) bool{
	key := crypto.CreateGroupkey(creator,1)

	if pt.SendTranPermi[key] == nil{

		//need check grop
		return pt.findGroupSendTxPerm(from)
	}

	if from == pt.SendTranPermi[key].Creator  {
		return true
	}
	if !pt.SendTranPermi[key].IsWhitListWork{
		if pt.SendTranPermi[key].BlackMembers.Manager != nil{


			for _ ,m := range pt.SendTranPermi[key].BlackMembers.Manager{
				//need check MemberID is grop id or not
				if pt.GropPermi[m.MemberID] != nil{
					return pt.findMember(m.MemberID,from,false)
				}else{
					if m.MemberID == from{return true}
				}

			}
		}




		//return false
	}else{
		if pt.SendTranPermi[key].WhiteMembers.Manager != nil{



			for _ ,m := range pt.SendTranPermi[key].WhiteMembers.Manager{
				if pt.GropPermi[m.MemberID] != nil{
					return pt.findMember(m.MemberID,from,true)
				}else{
					if m.MemberID == from{return true}
				}
			}
		}


	}
	return false
}
func (pt *PerminTable)checkSendTx(from,creator common.Address) bool{

	key := crypto.CreateGroupkey(creator,1)

	if pt.SendTranPermi[key] == nil{

		//need check grop
		return pt.findGroupSendTxPerm(from)
	}

	if from == pt.SendTranPermi[key].Creator  {
		return true
	}
	if !pt.SendTranPermi[key].IsWhitListWork{

		if pt.SendTranPermi[key].BlackMembers.Member != nil{


			for _ ,m := range pt.SendTranPermi[key].BlackMembers.Member{
				//need check MemberID is grop id or not
				if pt.GropPermi[m.MemberID] != nil{
					return pt.findMember(m.MemberID,from,false)
				}else{
					if m.MemberID == from{return false}
				}

			}
		}

		if pt.SendTranPermi[key].BlackMembers.Manager != nil{


		for _ ,m := range pt.SendTranPermi[key].BlackMembers.Manager{
			//need check MemberID is grop id or not
			if pt.GropPermi[m.MemberID] != nil{
				return pt.findMember(m.MemberID,from,false)
			}else{
				if m.MemberID == from{return true}
			}

		}
		}




		//return false
	}else{
		if pt.SendTranPermi[key].WhiteMembers.Manager != nil{



			for _ ,m := range pt.SendTranPermi[key].WhiteMembers.Manager{
				if pt.GropPermi[m.MemberID] != nil{
					return pt.findMember(m.MemberID,from,true)
				}else{
					if m.MemberID == from{return true}
				}
			}
		}

		if pt.SendTranPermi[key].WhiteMembers.Member != nil{
			for _ ,m := range pt.SendTranPermi[key].WhiteMembers.Member{
				if m.MemberID == from{return true}else{
					if pt.GropPermi[m.MemberID] != nil{
						return pt.findMember(m.MemberID,from,true)
					}


				}

			}
		}
	}
	return false
}




func (pt *PerminTable)findGroupSendTxPerm(member common.Address) (res bool)   {
	if pt.UserBasisPermi[member] == nil{
		return false
	}
	if len(pt.UserBasisPermi[member].GropList) > 0{
		for i :=0;i<len(pt.UserBasisPermi[member].GropList);i++{
			for j:=0;j<len(pt.RootList);j++{
				fmt.Println(pt.RootList[j])
				if pt.checkSendTx(pt.UserBasisPermi[member].GropList[i],pt.RootList[j]){
					return true;
				}
			}
		}
	}
	return false
}

func (pt *PerminTable)findMember(gropAddr,member common.Address,isWhitListWork bool) bool  {
	if !isWhitListWork{
		if pt.GropPermi[gropAddr] != nil{
			if pt.GropPermi[gropAddr].BlackMembers!=nil{
				if len(pt.GropPermi[gropAddr].BlackMembers.Member) >0 {
					totalN :=0
					for _,m := range pt.GropPermi[gropAddr].BlackMembers.Member{
						if pt.GropPermi[m.MemberID] != nil{
							if pt.findMember(m.MemberID,member,isWhitListWork){
								return false
							}
						}else{
							if m.MemberID == member{
								return false
							}
						}
						totalN++
					}
					if totalN == len(pt.GropPermi[gropAddr].BlackMembers.Member){
						return true
					}else{
						return false
					}
				}
				if len(pt.GropPermi[gropAddr].BlackMembers.Manager) >0 {
					totalN :=0
					for _,m := range pt.GropPermi[gropAddr].BlackMembers.Manager{
						if pt.GropPermi[m.MemberID] != nil{
							if pt.findMember(m.MemberID,member,isWhitListWork){
								return false
							}
						}else{
							if m.MemberID == member{
								return false
							}
						}
						totalN++
					}
					if totalN == len(pt.GropPermi[gropAddr].BlackMembers.Manager){
						return true
					}else{
						return false
					}
				}
			}
		}
	}else{
		if pt.GropPermi[gropAddr] != nil{
			if pt.GropPermi[gropAddr].WhiteMembers!=nil{
				if len(pt.GropPermi[gropAddr].WhiteMembers.Member) >0 {
					for _,m := range pt.GropPermi[gropAddr].WhiteMembers.Member{
						if pt.GropPermi[m.MemberID] != nil{
							if pt.findMember(m.MemberID,member,isWhitListWork){
								return true
							}
						}else{
							if m.MemberID == member{
								return true
							}
						}

					}
				}
				if len(pt.GropPermi[gropAddr].WhiteMembers.Manager) >0 {
					for _,ma := range pt.GropPermi[gropAddr].WhiteMembers.Manager{
						if pt.GropPermi[ma.MemberID] != nil{
							if pt.findMember(ma.MemberID,member,isWhitListWork){
								return true
							}
						}else{
							if ma.MemberID == member{
								return true
							}
						}

					}
				}
			}

		}else{
			if gropAddr == member{
				return true
			}
		}
	}

	return false
}


func (mt *MemberTable)clone() *MemberTable  {
	temp := &MemberTable{
		Manager:[]*MemberInfo{},
		Member:[]*MemberInfo{},
	}
	for _,manval :=range mt.Manager{
		temp.Manager = append(temp.Manager, manval)
	}

	for _,memval :=range mt.Member{
		temp.Member = append(temp.Manager, memval)
	}

	return temp

}


